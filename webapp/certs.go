// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"cloudeng.io/file"
)

// VerifyCertChain verifies the supplied certificate chain using the
// provided root certificates and verifies that the leaf certificate
// is valid for the specified dnsname.
// It returns the verified chains on success.
func VerifyCertChain(dnsname string, certs []*x509.Certificate, roots *x509.CertPool) ([][]*x509.Certificate, error) {
	opts := x509.VerifyOptions{
		DNSName: dnsname,
		Roots:   roots,
	}
	return verifyCertChainOpts(certs, opts)
}

// verifyCertChainOpts verifies the supplied certificate chain using the
// provided VerifyOptions but extracts the leaf and intermediates from the
// supplied certs slice.
// It returns the verified chains on success.
func verifyCertChainOpts(certs []*x509.Certificate, opts x509.VerifyOptions) ([][]*x509.Certificate, error) {
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates supplied")
	}
	var intermediates *x509.CertPool
	var leaf *x509.Certificate

	for _, cert := range certs {
		if cert.IsCA {
			if intermediates == nil {
				intermediates = x509.NewCertPool()
			}
			intermediates.AddCert(cert)
			continue
		}
		if leaf != nil {
			return nil, fmt.Errorf("expected exactly one leaf certificate, found multiple")
		}
		leaf = cert
	}
	if leaf == nil {
		return nil, fmt.Errorf("no leaf certificate found")
	}
	opts.Intermediates = intermediates
	return leaf.Verify(opts)
}

// ReadAndParseCertsPEM loads certificates from the specified PEM file.
func ReadAndParseCertsPEM(ctx context.Context, fs file.ReadFileFS, pemFile string) ([]*x509.Certificate, error) {
	data, err := fs.ReadFileCtx(ctx, pemFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read pem file %v: %v", pemFile, err)
	}
	certs, err := ParseCertsPEM(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate in %v: %v", pemFile, err)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in pem data")
	}
	return certs, nil
}

// ParseCertsPEM parses certificates from the provided PEM data.
func ParseCertsPEM(pemData []byte) ([]*x509.Certificate, error) {
	_, _, certsPEM := ParsePEM(pemData)
	certs, err := parseCertsPEM(certsPEM)
	if err != nil {
		return nil, err
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in pem data")
	}
	return certs, nil
}

func parseCertsPEM(certsPEM []*pem.Block) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for _, block := range certsPEM {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in pem data")
	}
	return certs, nil
}

// ParsePEM parses private keys and certificates from
// the provided PEM data.
func ParsePEM(pemData []byte) (privateKeys, publicKeys, certs []*pem.Block) {
	rest := pemData
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch {
		case block.Type == "CERTIFICATE":
			certs = append(certs, block)
		case strings.HasSuffix(block.Type, "PRIVATE KEY"):
			privateKeys = append(privateKeys, block)
		case strings.HasSuffix(block.Type, "PUBLIC KEY"):
			publicKeys = append(publicKeys, block)
		}
	}
	return
}

// ParsePrivateKeyDER parses a DER encoded private key.
// It tries PKCS#1, PKCS#8 and then SEC 1 for EC keys.
func ParsePrivateKeyDER(der []byte) (crypto.Signer, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, nil
		case *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	return nil, errors.New("failed to parse private key")
}

// ReadAndParsePrivateKeyPEM reads and parses a PEM encoded private key from the
// specified file.
func ReadAndParsePrivateKeyPEM(ctx context.Context, fs file.ReadFileFS, pemFile string) (crypto.Signer, error) {
	data, err := fs.ReadFileCtx(ctx, pemFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read pem file %v: %v", pemFile, err)
	}
	privateKeys, _, _ := ParsePEM(data)
	if len(privateKeys) == 0 {
		return nil, fmt.Errorf("no private key found in %v", pemFile)
	}
	if len(privateKeys) > 1 {
		return nil, fmt.Errorf("multiple private keys found in %v", pemFile)
	}
	return ParsePrivateKeyDER(privateKeys[0].Bytes)
}

// FindLeafPEM searches the supplied PEM blocks for the leaf certificate
// and returns its DER encoding along with the parsed x509.Certificate.
func FindLeafPEM(certsPEM []*pem.Block) ([]byte, *x509.Certificate, error) {
	for _, block := range certsPEM {
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, err
		}
		if !cert.IsCA {
			return block.Bytes, cert, nil
		}
	}
	return nil, nil, fmt.Errorf("no leaf certificate found")
}
