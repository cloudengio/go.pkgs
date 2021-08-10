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
	"cloudeng.io/cmdutil/subcmd"
)

type validateFlags struct {
	AllHosts           bool            `subcmd:"all,false,set to validate all of the hosts for a given DNS hostname or domain"`
	CustomROOTCA       string          `subcmd:"custom-root-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
	CheckSerialNumbers bool            `subcmd:"same-serial-numbers,true,check that all of the serial numbers for certs found on the same host are the same"`
	ValidFor           time.Duration   `subcmd:"valid-for,720h,check that the certifcates are valid for at least this duration"`
	Issuer             flags.Repeating `subcmd:"issuer,,check that the issuer urls match this regular expression"`
}

func validateCmd() *subcmd.Command {
	validateCmd := subcmd.NewCommand("validate", subcmd.MustRegisterFlagStruct(&validateFlags{}, nil, nil), validateCertificates, subcmd.AtLeastNArguments(1))
	validateCmd.Document(`validate the certificates for a host/domain`)

	return validateCmd
}

func validateCerts(ctx context.Context, cl *validateFlags, host string, addrs []string) error {
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
	cfg := customTLSConfig(pemfile)
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

func validateCertificates(ctx context.Context, values interface{}, args []string) error {
	cl := values.(*validateFlags)
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
