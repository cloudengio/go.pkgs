// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/file/localfs"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webauth/acme"
	"cloudeng.io/webapp/webauth/acme/certcache"
	goacme "golang.org/x/crypto/acme"
)

type revokeCmd struct{}

type RevokeCommonFlags struct {
	acme.ServiceFlags
	TestingCAPemFlag
	AccountKeyAliasFlag
	TLSCertStoreFlags
	awsconfig.AWSFlags
	IssuingCertPEMFile string `subcmd:"issuing-cert-pem,,'pem file containing the issuing certificate'"`
}

type revokeFlags struct {
	RevokeCommonFlags
	RevocationReason string `subcmd:"revocation-reason,,'the reason for revocation, one of unspecified,keyCompromise,affiliationChanged,superseded,cessationOfOperation,certificateHold'"`
	UseAccountKey    bool   `subcmd:"use-account-key,false,'use the acme account key to sign the revocation request'"`
}

type checkStatusFlags struct {
	RevokeCommonFlags
}

func (revokeCmd) revokeUsingKey(ctx context.Context, flags any, args []string) error {
	name := args[0]
	cl := flags.(*revokeFlags)

	issuer, err := getIssuer(ctx, cl.IssuingCertPEMFile)
	if err != nil {
		return fmt.Errorf("failed to read issuing certificate PEM file: %w", err)
	}

	reason, err := certcache.ParseRevocationReason(cl.RevocationReason)
	if err != nil {
		return err
	}

	cache, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags,
		certcache.WithSaveAccountKey(cl.AccountKeyAlias),
		certcache.WithReadonly(true))
	if err != nil {
		return err
	}

	acmeCfg := cl.AutocertConfig()
	accountKey, err := cache.GetAccountKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to obtain acme account key: %w", err)
	}

	httpClient, err := httpClientWithCustomCA(ctx, cl.TestingCAPem)
	if err != nil {
		return err
	}
	client := goacme.Client{
		DirectoryURL: acmeCfg.DirectoryURL(),
		HTTPClient:   httpClient,
		Key:          accountKey,
	}

	certPEM, err := cache.Get(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get certificate from cache: %w", err)
	}

	privKeys, _, certs := webapp.ParsePEM(certPEM)
	if err != nil {
		return fmt.Errorf("failed to parse certificate data: %w", err)
	}
	if len(certs) == 0 {
		return fmt.Errorf("no certificates found in certificate data")
	}
	if len(privKeys) == 0 {
		return fmt.Errorf("no private keys found in certificate data")
	}
	certKey, err := webapp.ParsePrivateKeyDER(privKeys[0].Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate private key: %w", err)
	}
	fmt.Printf("revoking certificate for %q\n", name)

	leafDER, _, err := webapp.FindLeafPEM(certs)
	if err != nil {
		return fmt.Errorf("failed to find leaf certificate: %w", err)
	}

	var revocationKey crypto.Signer
	if cl.UseAccountKey {
		revocationKey = certKey
	}
	err = client.RevokeCert(ctx, revocationKey, leafDER, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke certificate for %q: %w", name, err)
	}
	fmt.Printf("successfully revoked certificate for %q\n", name)

	status, err := getCertStatus(ctx, cache, name, issuer)
	if err != nil {
		return fmt.Errorf("failed to check OCSP status after revocation: %w", err)
	}
	if status != webapp.OCSPStatusRevoked {
		return fmt.Errorf("expected OCSP status 'revoked' after revocation, got %s", status)
	}
	return nil
}

func (revokeCmd) getStatus(ctx context.Context, flags any, args []string) error {
	cl := flags.(*checkStatusFlags)

	issuer, err := getIssuer(ctx, cl.IssuingCertPEMFile)
	if err != nil {
		return fmt.Errorf("failed to read issuing certificate PEM file: %w", err)
	}

	for _, name := range args {
		cache, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags,
			certcache.WithReadonly(true))
		if err != nil {
			return err
		}
		status, err := getCertStatus(ctx, cache, name, issuer)
		if err != nil {
			return fmt.Errorf("failed to check OCSP status: %w", err)
		}

		fmt.Printf("%q: %s\n", name, status)
	}
	// need issuer and leaf cert
	return nil
}

func getIssuer(ctx context.Context, filename string) (*x509.Certificate, error) {
	issuers, err := webapp.ReadAndParseCertsPEM(ctx, localfs.New(), filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read issuing certificate PEM file: %w", err)
	}
	if len(issuers) == 0 {
		return nil, fmt.Errorf("no issuing certificate found")
	}
	return issuers[0], nil
}
