// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jwtutil_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"cloudeng.io/webapp/webauth/jwtutil"
	"github.com/golang-jwt/jwt/v5"
)

func TestSignAndVerifyED25519(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key pair: %v", err)
	}

	keyID := "test-key-001"
	signer := jwtutil.NewED25519Signer(pub, priv, keyID)

	// 1. Test successful signing.
	claims := &jwt.RegisteredClaims{
		Subject:   "test-user",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	tokenString, err := signer.Sign(claims)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if len(tokenString) == 0 {
		t.Fatal("Sign() returned an empty token string")
	}

	// 2. Test successful verification with the same signer's KeyFunc.
	parsedClaims := &jwt.RegisteredClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenString, parsedClaims, signer.KeyFunc)
	if err != nil {
		t.Fatalf("ParseWithClaims failed: %v", err)
	}

	if !parsedToken.Valid {
		t.Error("token should be valid")
	}
	if got, want := parsedClaims.Subject, claims.Subject; got != want {
		t.Errorf("got subject %q, want %q", got, want)
	}
	if kid, ok := parsedToken.Header["kid"]; !ok || kid != keyID {
		t.Errorf("token header missing or has incorrect 'kid': got %v, want %v", kid, keyID)
	}

	// 3. Test successful verification with a separate PublicKeys instance.
	verifier := jwtutil.NewED25519PublicKey(pub, keyID)
	parsedToken2, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, verifier.KeyFunc)
	if err != nil {
		t.Fatalf("ParseWithClaims with separate verifier failed: %v", err)
	}
	if !parsedToken2.Valid {
		t.Error("token should be valid when using separate verifier")
	}
}

func TestKeyFuncErrors(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	keyID := "key-1"
	signer := jwtutil.NewED25519Signer(pub, priv, keyID)
	claims := &jwt.RegisteredClaims{Subject: "test"}

	t.Run("Wrong Key ID", func(t *testing.T) {
		tokenString, err := signer.Sign(claims)
		if err != nil {
			t.Fatal(err)
		}

		// Create a verifier with the correct public key but the wrong ID.
		verifier := jwtutil.NewED25519PublicKey(pub, "key-2")
		_, err = jwt.Parse(tokenString, verifier.KeyFunc)
		if !errors.Is(err, jwt.ErrTokenUnverifiable) {
			t.Errorf("expected jwt.ErrTokenUnverifiable, got %v", err)
		}
	})

	t.Run("Wrong Signing Method", func(t *testing.T) {
		// Create a token with a different signing method.
		hmacToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		hmacString, err := hmacToken.SignedString([]byte("dummy-secret"))
		if err != nil {
			t.Fatal(err)
		}

		// Attempt to verify it with the Ed25519 KeyFunc.
		_, err = jwt.Parse(hmacString, signer.KeyFunc)
		if !errors.Is(err, jwt.ErrSignatureInvalid) {
			t.Errorf("expected jwt.ErrSignatureInvalid, got %v", err)
		}
	})

	t.Run("Missing Key ID in Token", func(t *testing.T) {
		// Manually create a token without setting the 'kid' header.
		token := jwt.NewWithClaims(&jwt.SigningMethodEd25519{}, claims)
		tokenString, err := token.SignedString(priv)
		if err != nil {
			t.Fatal(err)
		}

		_, err = jwt.Parse(tokenString, signer.KeyFunc)
		if !errors.Is(err, jwt.ErrTokenUnverifiable) {
			t.Errorf("expected jwt.ErrInvalidKey for missing 'kid', got %v", err)
		}
	})
}
