// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jwtutil

import (
	"crypto/ed25519"

	"github.com/golang-jwt/jwt/v5"
)

// PublicKeys represents an interface to the jwt.KeyFunc called by the JWT
// parser to retrieve the public key or keys used for verifying JWTs.
type PublicKeys interface {
	KeyFunc(token *jwt.Token) (any, error)
}

// NewED25519PublicKey creates a new ED25519PublicKey instance with the given public key and key ID.
type ED25519PublicKey struct {
	pub ed25519.PublicKey
	id  string
}

func NewED25519PublicKey(pub ed25519.PublicKey, id string) *ED25519PublicKey {
	return &ED25519PublicKey{
		pub: pub,
		id:  id,
	}
}

func (v ED25519PublicKey) KeyFunc(token *jwt.Token) (any, error) {
	if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
		return nil, jwt.ErrSignatureInvalid
	}
	if token.Header["kid"] != v.id {
		return nil, jwt.ErrInvalidKey
	}
	return v.pub, nil
}
