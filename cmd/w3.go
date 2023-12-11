package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/multiformats/go-multihash"
	"github.com/urfave/cli/v2"
	"github.com/web3-storage/go-ucanto/core/car"
	"github.com/web3-storage/go-ucanto/core/delegation"
	"github.com/web3-storage/go-w3up/capability/storeadd"
	"github.com/web3-storage/go-w3up/capability/uploadadd"
	"github.com/web3-storage/go-w3up/capability/uploadlist"
	"github.com/web3-storage/go-w3up/client"
)

func main() {
	app := &cli.App{
		Name:  "w3",
		Usage: "interact with the web3.storage API",
		Commands: []*cli.Command{
			{
				Name:   "whoami",
				Usage:  "Print information about the current agent.",
				Action: whoami,
			},
			{
				Name:    "up",
				Aliases: []string{"upload"},
				Usage:   "Store a file(s) to the service and register an upload.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "space",
						Value: "",
						Usage: "DID of space to list uploads from.",
					},
					&cli.StringFlag{
						Name:  "proof",
						Value: "",
						Usage: "Path to file containing UCAN proof(s) for the operation.",
					},
					&cli.StringFlag{
						Name:    "car",
						Aliases: []string{"c"},
						Value:   "",
						Usage:   "Path to CAR file to upload (max 4GB).",
					},
				},
				Action: up,
			},
			{
				Name:    "ls",
				Aliases: []string{"list"},
				Usage:   "List uploads in the current space.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "space",
						Value: "",
						Usage: "DID of space to list uploads from.",
					},
					&cli.StringFlag{
						Name:  "proof",
						Value: "",
						Usage: "Path to file containing UCAN proof(s) for the operation.",
					},
					&cli.BoolFlag{
						Name:  "shards",
						Value: false,
						Usage: "Display shard CID(s) for each upload root.",
					},
				},
				Action: ls,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func whoami(cCtx *cli.Context) error {
	s := mustGetSigner()
	fmt.Println(s.DID())
	return nil
}

func up(cCtx *cli.Context) error {
	signer := mustGetSigner()
	conn := mustGetConnection()
	space := mustParseDID(cCtx.String("space"))
	proof := mustGetProof(cCtx.String("proof"))

	f0, err := os.Open(cCtx.String("car"))
	if err != nil {
		log.Fatalf("opening file: %s", err)
	}

	stat, err := f0.Stat()
	if err != nil {
		log.Fatalf("stat file: %s", err)
	}

	mh, err := multihash.SumStream(f0, multihash.SHA2_256, -1)
	if err != nil {
		log.Fatalf("hashing CAR: %s", err)
	}

	err = f0.Close()
	if err != nil {
		log.Fatalf("closing file: %s", err)
	}

	link := cidlink.Link{Cid: cid.NewCidV1(0x0202, mh)}
	fmt.Println(link.String())

	rcpt, err := client.StoreAdd(
		signer,
		space,
		&storeadd.Caveat{
			Link: link,
			Size: uint64(stat.Size()),
		},
		client.WithConnection(conn),
		client.WithProofs([]delegation.Delegation{proof}),
	)
	if err != nil {
		return err
	}

	if rcpt.Out().Error() != nil {
		log.Fatalf("%+v\n", rcpt.Out().Error())
	}

	if rcpt.Out().Ok().Status == "upload" {
		f2, err := os.Open(cCtx.String("car"))
		if err != nil {
			log.Fatalf("opening file: %s", err)
		}

		fmt.Println(*rcpt.Out().Ok().Url)
		hr, err := http.NewRequest("PUT", *rcpt.Out().Ok().Url, f2)
		if err != nil {
			log.Fatalf("creating HTTP request: %s", err)
		}

		hdr := map[string][]string{}
		for k, v := range rcpt.Out().Ok().Headers.Values {
			fmt.Printf("%s: %s\n", k, v)
			hdr[k] = []string{v}
		}

		hr.Header = hdr
		hr.ContentLength = stat.Size()
		httpClient := http.Client{}
		res, err := httpClient.Do(hr)
		if err != nil {
			log.Fatalf("doing HTTP request: %s", err)
		}
		if res.StatusCode != 200 {
			log.Fatalf("non-200 status code while uploading file: %d", res.StatusCode)
		}
		err = res.Body.Close()
		if err != nil {
			log.Fatalf("closing request body: %s", err)
		}
	}

	f3, err := os.Open(cCtx.String("car"))
	if err != nil {
		log.Fatalf("opening file: %s", err)
	}
	roots, _, err := car.Decode(f3)
	if err != nil {
		log.Fatalf("reading roots: %s", err)
	}
	err = f3.Close()
	if err != nil {
		log.Fatalf("closing file: %s", err)
	}
	if len(roots) > 0 {
		rcpt, err := client.UploadAdd(
			signer,
			space,
			&uploadadd.Caveat{
				Root:   roots[0],
				Shards: []ipld.Link{link},
			},
			client.WithConnection(conn),
			client.WithProofs([]delegation.Delegation{proof}),
		)
		if err != nil {
			return err
		}
		if rcpt.Out().Error() != nil {
			log.Fatalf("%+v\n", rcpt.Out().Error())
		}

		fmt.Printf("⁂ https://w3s.link/ipfs/%s\n", roots[0])
	}

	return nil
}

func ls(cCtx *cli.Context) error {
	signer := mustGetSigner()
	conn := mustGetConnection()
	space := mustParseDID(cCtx.String("space"))
	proof := mustGetProof(cCtx.String("proof"))

	rcpt, err := client.UploadList(
		signer,
		space,
		&uploadlist.Caveat{},
		client.WithConnection(conn),
		client.WithProofs([]delegation.Delegation{proof}),
	)
	if err != nil {
		return err
	}

	if rcpt.Out().Error() != nil {
		log.Fatalf("%+v\n", rcpt.Out().Error())
	}

	for _, r := range rcpt.Out().Ok().Results {
		fmt.Printf("%s\n", r.Root)
		if cCtx.Bool("shards") {
			for _, s := range r.Shards {
				fmt.Printf("\t%s\n", s)
			}
		}
	}

	return nil
}
