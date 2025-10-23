// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"regexp"
	"time"

	"cloudeng.io/cmdutil/flags"
)

type ValidateFlags struct {
	CustomROOTCA       string          `subcmd:"custom-root-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
	CheckSerialNumbers bool            `subcmd:"same-serial-numbers,true,check that all of the serial numbers for certs found on the same host are the same"`
	ValidFor           time.Duration   `subcmd:"valid-for,720h,check that the certifcates are valid for at least this duration"`
	Issuer             flags.Repeating `subcmd:"issuer,,check that the issuer urls match this regular expression"`
}

type validateFileFlags struct {
	ValidateFlags
}

type validateHostFlags struct {
	ValidateFlags
	AllHosts bool `subcmd:"all,false,set to validate all of the hosts for a given DNS hostname or domain"`
}

func validateCerts(ctx context.Context, cl *validateHostFlags, host string, addrs []string) error {
	var serial *big.Int
	var serialFrom string
	expiry := time.Now().Add(cl.ValidFor)

	regexps := []*regexp.Regexp{}
	for _, expr := range cl.Issuer.Values {
		re, err := regexp.Compile(expr)
		if err != nil {
			return err
		}
		regexps = append(regexps, re)
	}
	for _, addr := range addrs {
		certs, err := downloadCert(ctx, cl.CustomROOTCA, host, addr)
		if err != nil {
			return err
		}
		cert0 := certs[0]
		if cl.CheckSerialNumbers {
			if serial == nil {
				serial = cert0.SerialNumber
				serialFrom = addr
			} else if serial.Cmp(cert0.SerialNumber) != 0 {
				return fmt.Errorf("%v: mismatched serial numbers: (%v: %v) != (%v: %v)", host, serialFrom, serial, addr, cert0.SerialNumber)
			}
		}
		if cert0.NotAfter.Before(expiry) {
			return fmt.Errorf("%v: %v: cert expires before (%v before %v", host, addr, cert0.NotAfter, expiry)
		}
		for _, re := range regexps {
			found := false
			for _, url := range cert0.IssuingCertificateURL {
				if re.MatchString(url) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%v: %v: %v is not in the list of issuer urls", host, addr, re)
			}
		}
		fmt.Printf("%v: %v: ok\n", host, addr)
		fmt.Printf("\tserial:      %v\n", cert0.SerialNumber)
		fmt.Printf("\tvalid until: %v\n", cert0.NotAfter)
		for _, url := range cert0.IssuingCertificateURL {
			fmt.Printf("\tissuer:  %v\n", url)
		}
	}
	return nil
}

func ignoreIPv6(addrs []string) []string {
	ipv4 := []string{}
	for _, addr := range addrs {
		if len(net.ParseIP(addr).To4()) == net.IPv4len {
			ipv4 = append(ipv4, addr)
		}
	}
	return ipv4
}

func downloadCert(ctx context.Context, pemfile, host, addr string) ([]*x509.Certificate, error) {
	cfg, err := customTLSConfig(ctx, pemfile)
	if err != nil {
		return nil, err
	}
	cfg.ServerName = host
	conn, err := tls.Dial("tcp", net.JoinHostPort(addr, "443"), cfg)
	if err != nil {
		return nil, err
	}
	cs := conn.ConnectionState()
	if len(cs.PeerCertificates) > 0 {
		return conn.ConnectionState().PeerCertificates, nil
	}
	return nil, fmt.Errorf("no peer certificates found for host %v @ %v", host, addr)

}

func expandHost(host string, expand bool) ([]string, error) {
	if expand {
		return net.LookupHost(host)
	}
	return []string{host}, nil
}

func (_ certsCmd) validateHostCertificates(ctx context.Context, values any, args []string) error {
	cl := values.(*validateHostFlags)
	for _, host := range args {
		all, err := expandHost(host, cl.AllHosts)
		if err != nil {
			return err
		}
		ipv4Only := ignoreIPv6(all)
		if err := validateCerts(ctx, cl, host, ipv4Only); err != nil {
			return err
		}
	}

	return nil
}

func certificatesFromPEM(pemFile string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read pem file %v: %v", pemFile, err)
	}
	certs := []*x509.Certificate{}
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate in %v: %v", pemFile, err)
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in pem data")
	}
	return certs, nil
}

func (_ certsCmd) validatePEMFile(_ context.Context, cl *validateFileFlags, pemFile string) error {
	certs, err := certificatesFromPEM(pemFile)
	if err != nil {
		return err
	}
	var leaves []*x509.Certificate
	var intermediates, root *x509.CertPool

	intermediates = x509.NewCertPool()
	for _, cert := range certs {
		if cert.IsCA {
			intermediates.AddCert(cert)
			continue
		}
		leaves = append(leaves, cert)
	}
	if len(leaves) != 1 {
		return fmt.Errorf("%v: expected exactly one leaf certificate, found %v", pemFile, len(leaves))
	}
	leaf := leaves[0]

	if cl.CustomROOTCA != "" {
		rootCerts, err := certificatesFromPEM(cl.CustomROOTCA)
		if err != nil {
			return fmt.Errorf("failed to load custom root CA from %v: %v", cl.CustomROOTCA, err)
		}
		root = x509.NewCertPool()
		for _, rc := range rootCerts {
			root.AddCert(rc)
		}
	}

	for _, ic := range leaf.IssuingCertificateURL {
		fmt.Printf("leaf issuer url: %v\n", ic)
	}
	opts := x509.VerifyOptions{
		Roots:         root,
		Intermediates: intermediates,
		CurrentTime:   time.Now(),
	}
	if _, err := leaf.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed for %v: %v", pemFile, err)
	}
	expiry := time.Now().Add(cl.ValidFor)
	if leaf.NotAfter.Before(expiry) {
		return fmt.Errorf("%v: cert expires before (%v before %v", pemFile, leaf.NotAfter, expiry)
	}
	return nil
}

func (c certsCmd) validatePEMFiles(ctx context.Context, values any, args []string) error {
	cl := values.(*validateFileFlags)
	for _, pemFile := range args {
		if err := c.validatePEMFile(ctx, cl, pemFile); err != nil {
			return err
		}
		fmt.Printf("%v: ok\n", pemFile)
	}
	return nil
}
