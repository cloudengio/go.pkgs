// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package auth0

import (
	"fmt"
	"net/http"
	"net/url"

	jose "github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/json"
	"github.com/square/go-jose/v3/jwt"
)

const jwksEndpoint = "/.well-known/jwks.json"

// JWKS represents the KWT Key Set returned by auth0.com.
// See https://auth0.com/docs/tokens/json-web-tokens/json-web-key-set-properties
type JWKS struct {
	*jose.JSONWebKeySet
}

type Option func(*Authenticator)

func RS256() Option {
	return func(a *Authenticator) {
		a.algo = "RS256"
	}
}

func StaticJWKS(jwks *JWKS) Option {
	return func(a *Authenticator) {
		a.staticJWKS = jwks
	}
}

func ensureHTTPS(raw string) string {
	u, _ := url.Parse(raw)
	u.Scheme = "https"
	return u.String()
}

func NewAuthenticator(domain, audience string, opts ...Option) (*Authenticator, error) {
	a := &Authenticator{
		domain:   ensureHTTPS(domain),
		audience: ensureHTTPS(audience),
		cn:       "CN=" + domain,
	}
	for _, fn := range opts {
		fn(a)
	}
	if len(a.algo) == 0 {
		RS256()(a)
	}
	return a, a.refresh()
}

func (a *Authenticator) refresh() error {
	if a.staticJWKS != nil {
		a.jwks = a.staticJWKS
		return nil
	}
	jwks, err := JWKSForDomain(a.domain)
	if err != nil {
		return err
	}
	keys := []jose.JSONWebKey{}
	a.jwks = jwks
	for _, key := range jwks.Keys {
		if key.Algorithm != a.algo {
			continue
		}
		if key.Use != "sig" {
			continue
		}
		if len(key.Certificates) == 0 {
			continue
		}
		if key.Certificates[0].Issuer.String() != a.cn {
			continue
		}
		keys = append(keys, key)
	}
	a.jwks.Keys = keys
	return nil
}

type Authenticator struct {
	domain, audience string
	algo             string
	cn               string
	staticJWKS       *JWKS
	jwks             *JWKS
}

func (a *Authenticator) CheckJWT(token string) error {
	tok, err := jwt.ParseSigned(token)
	if err != nil {
		return fmt.Errorf("failed to parse jwt: %v", err)
	}
	claims := jwt.Claims{}
	if err := tok.Claims(a.jwks.JSONWebKeySet, &claims); err != nil {
		return err
	}
	expected := jwt.Expected{
		Issuer:   a.domain,
		Audience: []string{a.audience},
	}
	return claims.Validate(expected)
}

func JWKSForDomain(tenant string) (*JWKS, error) {
	endpoint := ensureHTTPS(tenant) + jwksEndpoint
	resp, err := http.Get(endpoint) // #nosec G107
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to access %v: %v", endpoint, resp.StatusCode)
	}
	defer resp.Body.Close()
	var jwks = &JWKS{}
	if err := json.NewDecoder(resp.Body).Decode(jwks); err != nil {
		return nil, fmt.Errorf("failed to decode response from %v: %v", endpoint, err)
	}
	return jwks, nil
}
