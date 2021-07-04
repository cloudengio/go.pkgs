// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package auth0_test

import (
	"bytes"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/webapp/webauth/auth0"
	"github.com/square/go-jose/v3/json"
)

func loadJWKS(t *testing.T, filename string) *auth0.JWKS {
	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	rd := bytes.NewBuffer(raw)
	jwks := &auth0.JWKS{}
	if err := json.NewDecoder(rd).Decode(jwks); err != nil {
		t.Fatal(err)
	}
	return jwks
}

func TestCertCompatibility(t *testing.T) {
	jwks := loadJWKS(t, filepath.Join("testdata", "example_key.json"))
	validateKey(t, jwks, "woFxZlV8zDjgRIkul-O30")
	validateKey(t, jwks, "Lw_0LqJxnpPyFFnGmrv-3")

}

func validateKey(t *testing.T, jwks *auth0.JWKS, id string) {
	keys := jwks.Key(id)
	if got, want := len(keys), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	key := keys[0]
	if got, want := key.KeyID, id; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := key.Algorithm, "RS256"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	cert := key.Certificates[0]
	if got, want := cert.Issuer.String(), "CN=dev-58g8bjfp.us.auth0.com"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	pk := cert.PublicKey
	if got, ok := pk.(*rsa.PublicKey); !ok {
		t.Errorf("wrong public key type: %T", got)
	}
}

func TestToken(t *testing.T) {
	tok, err := os.ReadFile(filepath.Join("testdata", "sample-token.txt"))
	if err != nil {
		t.Fatal(err)
	}
	jwks := loadJWKS(t, filepath.Join("testdata", "example_key.json"))
	checker, err := auth0.NewAuthenticator(
		"dev-58g8bjfp.us.auth0.com/",
		"dev-58g8bjfp.us.auth0.com/api/v2/",
		auth0.StaticJWKS(jwks),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := checker.CheckJWT(string(tok)); err != nil {
		t.Fatal(err)
	}
	for i, invalid := range []struct {
		tok    []byte
		errmsg string
	}{
		{
			[]byte{},
			"failed to parse jwt: square/go-jose: compact JWS format must have three parts",
		},
		{
			[]byte{'a'},
			"failed to parse jwt: square/go-jose: compact JWS format must have three parts",
		},
		{
			append([]byte{'a'}, tok...),
			"failed to parse jwt: illegal base64 data at input byte 76",
		},
		{
			append(append([]byte{}, tok...), 'z'),
			"square/go-jose: error in cryptographic primitive",
		},
		{
			append(append(append([]byte{}, tok[:3]...), 'z'), tok[4:]...),
			"square/go-jose: error in cryptographic primitive",
		},
	} {
		err := checker.CheckJWT(string(invalid.tok) + "x")
		if err == nil || err.Error() != invalid.errmsg {
			t.Errorf("%v: missing or unexpected error: __%v__", i, err.Error())
		}
	}
}
