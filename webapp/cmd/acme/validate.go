// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/file/localfs"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/tlsvalidate"
)

type ValidateFlags struct {
	TLSPort            string          `subcmd:"tls-port,443,the TLS port to use when validating hosts"`
	CustomROOTCA       string          `subcmd:"custom-root-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
	CheckSerialNumbers bool            `subcmd:"same-serial-numbers,true,check that all of the serial numbers for certs found on the same host are the same"`
	ValidFor           time.Duration   `subcmd:"valid-for,360h,check that the certifcates are valid for at least this duration"`
	Issuer             flags.Repeating `subcmd:"issuer,,check that the issuer urls match this regular expression"`
}

type validateFileFlags struct {
	ValidateFlags
}

type validateHostFlags struct {
	ValidateFlags
	AllHosts bool `subcmd:"all,false,set to validate all of the hosts for a given DNS hostname or domain"`
}

func (certsCmd) validateHostCertificatesCmd(ctx context.Context, values any, args []string) error {
	cl := values.(*validateHostFlags)
	for _, host := range args {
		if err := validateCerts(ctx, cl, host); err != nil {
			return err
		}
		fmt.Printf("%v: ok\n", host)
	}
	return nil
}

func validateCerts(ctx context.Context, cl *validateHostFlags, host string) error {
	regexps := []*regexp.Regexp{}
	for _, expr := range cl.Issuer.Values {
		re, err := regexp.Compile(expr)
		if err != nil {
			return err
		}
		regexps = append(regexps, re)
	}
	opts := []tlsvalidate.Option{
		tlsvalidate.WithExpandDNSNames(cl.AllHosts),
		tlsvalidate.WithCheckSerialNumbers(cl.CheckSerialNumbers),
		tlsvalidate.WithValidForAtLeast(cl.ValidFor),
		tlsvalidate.WithIssuerRegexps(regexps...),
	}
	if len(cl.CustomROOTCA) > 0 {
		root, err := devtest.CertPoolForTesting(cl.CustomROOTCA)
		if err != nil {
			return fmt.Errorf("failed to obtain cert pool containing %v: %w", cl.CustomROOTCA, err)
		}
		opts = append(opts, tlsvalidate.WithRootCAs(root))
	}

	validator := tlsvalidate.NewValidator(opts...)
	return validator.Validate(ctx, host, cl.TLSPort)
}

func (certsCmd) validatePEMFile(ctx context.Context, pemFile, rootCA string) error {
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

func (c certsCmd) validatePEMFilesCmd(ctx context.Context, values any, args []string) error {
	cl := values.(*validateFileFlags)
	for _, pemFile := range args {
		if err := c.validatePEMFile(ctx, pemFile, cl.CustomROOTCA); err != nil {
			return err
		}
		fmt.Printf("%v: ok\n", pemFile)
	}
	return nil
}
