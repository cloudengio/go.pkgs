// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"time"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/file/localfs"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devtest"
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

func (_ certsCmd) validateHostCertificatesCmd(ctx context.Context, values any, args []string) error {
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

	tlsCfg := &tls.Config{}
	root, err := devtest.CertPoolForTesting(cl.CustomROOTCA)
	if err != nil {
		return fmt.Errorf("failed to obtain cert pool containing %v: %w", cl.CustomROOTCA, err)
	}
	tlsCfg.RootCAs = root

	for _, addr := range addrs {
		certs, err := downloadCert(ctx, tlsCfg, host, addr)
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

func downloadCert(ctx context.Context, cfg *tls.Config, host, addr string) ([]*x509.Certificate, error) {
	cfg = cfg.Clone()
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

func (_ certsCmd) validatePEMFile(ctx context.Context, cl *validateFileFlags, pemFile, rootCA string) error {
	certs, err := webapp.ReadAndParseCertsPEM(ctx, localfs.New(), pemFile)
	if err != nil {
		return err
	}
	root, err := devtest.CertPoolForTesting(rootCA)
	if err != nil {
		return fmt.Errorf("failed to obtain cert pool containing %v: %w", rootCA, err)
	}
	_, err = webapp.VerifyCertChain("", certs, root)
	return err
}

func (c certsCmd) validatePEMFiles(ctx context.Context, values any, args []string) error {
	cl := values.(*validateFileFlags)
	for _, pemFile := range args {
		if err := c.validatePEMFile(ctx, cl, pemFile, cl.CustomROOTCA); err != nil {
			return err
		}
		fmt.Printf("%v: ok\n", pemFile)
	}
	return nil
}
