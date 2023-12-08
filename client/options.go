package client

import (
	"github.com/web3-storage/go-ucanto/client"
	"github.com/web3-storage/go-ucanto/core/delegation"
	"github.com/web3-storage/go-ucanto/ucan"
)

// Option is an option configuring a UCAN delegation.
type Option func(cfg *ClientConfig) error

type ClientConfig struct {
	conn client.Connection
	exp  uint64
	nbf  uint64
	nnc  string
	fct  []ucan.FactBuilder
	prf  []delegation.Delegation
}

// WithConnection configures the connection to execute the invocation on.
func WithConnection(conn client.Connection) Option {
	return func(cfg *ClientConfig) error {
		cfg.conn = conn
		return nil
	}
}

// WithExpiration configures the expiration time in UTC seconds since Unix
// epoch. Set this to -1 for no expiration.
func WithExpiration(exp uint64) Option {
	return func(cfg *ClientConfig) error {
		cfg.exp = exp
		return nil
	}
}

// WithNotBefore configures the time in UTC seconds since Unix epoch when the
// UCAN will become valid.
func WithNotBefore(nbf uint64) Option {
	return func(cfg *ClientConfig) error {
		cfg.nbf = nbf
		return nil
	}
}

// WithNonce configures the nonce value for the UCAN.
func WithNonce(nnc string) Option {
	return func(cfg *ClientConfig) error {
		cfg.nnc = nnc
		return nil
	}
}

// WithFacts configures the facts for the UCAN.
func WithFacts(fct []ucan.FactBuilder) Option {
	return func(cfg *ClientConfig) error {
		cfg.fct = fct
		return nil
	}
}

// WithProofs configures the proofs for the UCAN. If the `issuer` of this
// `Delegation` is not the resource owner / service provider, for the delegated
// capabilities, the `proofs` must contain valid `Proof`s containing
// delegations to the `issuer`.
func WithProofs(prf []delegation.Delegation) Option {
	return func(cfg *ClientConfig) error {
		cfg.prf = prf
		return nil
	}
}

func convertToInvocationOptions(cfg ClientConfig) []delegation.Option {
	var opts []delegation.Option
	if cfg.exp != 0 {
		opts = append(opts, delegation.WithExpiration(cfg.exp))
	}
	if cfg.nbf != 0 {
		opts = append(opts, delegation.WithNotBefore(cfg.nbf))
	}
	if cfg.nnc != "" {
		opts = append(opts, delegation.WithNonce(cfg.nnc))
	}
	if cfg.fct != nil {
		opts = append(opts, delegation.WithFacts(cfg.fct))
	}
	if cfg.prf != nil {
		opts = append(opts, delegation.WithProofs(cfg.prf))
	}
	return opts
}
