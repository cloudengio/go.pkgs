// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package tlsvalidate provides functions for validating TLS certificates
// across multiple hosts and addresses.
package tlsvalidate

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

// Option represents an option for configuring a Validator.
type Option func(o *options)

// WithIPv4Only returns an option that configures the validator to only consider
// IPv4 addresses for a host.
func WithIPv4Only(ipv4Only bool) Option {
	return func(o *options) {
		o.ipv4Only = ipv4Only
	}
}

// WithValidForAtLeast returns an option that configures the validator to check
// that the certificate is valid for at least the specified duration.
func WithValidForAtLeast(validFor time.Duration) Option {
	return func(o *options) {
		o.validFor = validFor
	}
}

// WithIssuerRegexps returns an option that configures the validator to check
// that the certificate's issuer matches at least one of the provided regular
// expressions.
func WithIssuerRegexps(exprs ...*regexp.Regexp) Option {
	return func(o *options) {
		o.issuerREs = exprs
	}
}

// WithExpandDNSNames returns an option that configures the validator to expand
// the supplied hostname to all of its IP addresses. If false, the hostname
// is used as is.
func WithExpandDNSNames(expand bool) Option {
	return func(o *options) {
		o.expand = expand
	}
}

// WithRootCAs returns an option that configures the validator to use the
// supplied pool of root CAs for verification.
func WithRootCAs(rootCAs *x509.CertPool) Option {
	return func(o *options) {
		o.rootCAs = rootCAs
	}
}

// WithCheckSerialNumbers returns an option that configures the validator to
// check that the certificates for all IP addresses for a given host have the
// same serial number.
func WithCheckSerialNumbers(check bool) Option {
	return func(o *options) {
		o.checkSerial = check
	}
}

// WithTLSMinVersion returns an option that configures the validator to check
// that the TLS version used is at least the specified version.
func WithTLSMinVersion(version uint16) Option {
	return func(o *options) {
		o.tlsMinVer = version
	}
}

// WithCiphersuites returns an option that configures the validator to check
// that the ciphersuite used is one of the specified ciphersuites.
func WithCiphersuites(suites []uint16) Option {
	return func(o *options) {
		o.ciphersuites = suites
	}
}

type options struct {
	ipv4Only     bool
	validFor     time.Duration
	issuerREs    []*regexp.Regexp
	expand       bool
	rootCAs      *x509.CertPool
	checkSerial  bool
	tlsMinVer    uint16
	ciphersuites []uint16
}

// Validator provides a way to validate TLS certificates.
type Validator struct {
	opts options
}

// NewValidator returns a new Validator configured with the supplied options.
func NewValidator(opts ...Option) *Validator {
	v := &Validator{}
	for _, opt := range opts {
		opt(&v.opts)
	}
	return v
}

// Validate performs TLS validation for the given host and port. It may expand
// the host to multiple IP addresses and will validate each one concurrently.
func (v *Validator) Validate(ctx context.Context, host, port string) error {
	addrs, err := v.expandHost(host)
	if err != nil {
		return err
	}
	addrs = v.ignoreIPv6(addrs)
	state := &tlsStates{
		states: make([]tlsState, 0, len(addrs)),
	}
	g, ctx := errgroup.WithContext(ctx)
	for _, addr := range addrs {
		g.Go(func() error {
			s, err := v.getTLSState(ctx, &tls.Config{ //nolint:gosec // G402 we want to test min version handling
				ServerName:    host,
				RootCAs:       v.opts.rootCAs,
				MinVersion:    v.opts.tlsMinVer,
				CipherSuites:  v.opts.ciphersuites,
				Renegotiation: tls.RenegotiateNever,
			}, addr, port)
			if err != nil {
				return err
			}
			state.add(tlsState{
				host:  host,
				addr:  addr,
				port:  port,
				state: s,
			})
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	var errs errors.M
	var serial *big.Int
	for _, cs := range state.states {
		if err := v.validateConnectionState(cs.state); err != nil {
			errs.Append(fmt.Errorf("host %q port %q: %w", cs.host, cs.port, err))
		}
		if v.opts.checkSerial {
			if serial == nil {
				serial = cs.state.PeerCertificates[0].SerialNumber
				continue
			}
			if serial.Cmp(cs.state.PeerCertificates[0].SerialNumber) != 0 {
				errs.Append(fmt.Errorf("%v: %v mismatched serial numbers: (%v) != (%v)", host, cs.addr, serial, cs.state.PeerCertificates[0].SerialNumber))
			}
		}
	}
	return errs.Err()
}

func (v *Validator) validateConnectionState(state tls.ConnectionState) error {
	leaf := state.PeerCertificates[0]
	if len(v.opts.issuerREs) > 0 {
		matched := false
		issuer := leaf.Issuer.String()
		for _, re := range v.opts.issuerREs {
			if re.MatchString(issuer) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("certificate issuer %q does not match any of the specified patterns", issuer)
		}
	}
	if v.opts.validFor > 0 {
		if validFor := time.Until(leaf.NotAfter); validFor < v.opts.validFor {
			return fmt.Errorf("certificate is valid for %v which is less than the required %v", validFor, v.opts.validFor)
		}
	}
	return nil
}

func (v *Validator) ignoreIPv6(addrs []string) []string {
	if !v.opts.ipv4Only {
		return addrs
	}
	ipv4 := []string{}
	for _, addr := range addrs {
		if len(net.ParseIP(addr).To4()) == net.IPv4len {
			ipv4 = append(ipv4, addr)
		}
	}
	return ipv4
}

func (v *Validator) expandHost(host string) ([]string, error) {
	if v.opts.expand {
		return net.LookupHost(host)
	}
	return []string{host}, nil
}

func (v *Validator) getTLSState(ctx context.Context, cfg *tls.Config, addr, port string) (tls.ConnectionState, error) {
	cfg = cfg.Clone()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(addr, port))
	if err != nil {
		return tls.ConnectionState{}, err
	}
	defer conn.Close()
	tlsConn := tls.Client(conn, cfg)
	defer tlsConn.Close()
	if err := tlsConn.Handshake(); err != nil {
		return tls.ConnectionState{}, err
	}
	return tlsConn.ConnectionState(), nil
}

type tlsState struct {
	host, addr, port string
	state            tls.ConnectionState
}

type tlsStates struct {
	mu     sync.Mutex
	states []tlsState
}

func (ts *tlsStates) add(state tlsState) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.states = append(ts.states, state)
}
