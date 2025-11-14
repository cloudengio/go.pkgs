// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jwtutil provides support for creating and verifying JSON
// Web Tokens (JWTs) managed by the github.com/golang-jwt/jwt/v5
// package. This package provides simplified wrappers around the
// JWT signing and verification process to allow for more convenient
// usage in web applications.
package jwtutil

import (
	"crypto/ed25519"

	"github.com/golang-jwt/jwt/v5"
)

type Signer interface {
	Sign(jwt.Claims) (string, error)
	PublicKeys
}

// ED25519Signer implements the Signer interface using an Ed25519 private key.
type ED25519Signer struct {
	ED25519PublicKey
	priv ed25519.PrivateKey
}

func NewED25519Signer(pub ed25519.PublicKey, priv ed25519.PrivateKey, id string) ED25519Signer {
	return ED25519Signer{
		priv: priv,
		ED25519PublicKey: ED25519PublicKey{
			pub: pub,
			id:  id,
		},
	}
}
func (s ED25519Signer) Sign(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, claims)
	token.Header["kid"] = s.id // Set the key ID in the header
	return token.SignedString(s.priv)
}
