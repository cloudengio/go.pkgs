// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package hash_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/algo/hash"
)

const (
	helloWorld = "hello world"

	// base64 encoded digests for "hello world"
	md5HelloWorldB64    = "XrY7u+Ae7tCTyyK7j1rNww=="
	sha1HelloWorldB64   = "Kq5sNclPz7QV2+lfQIuc6R7oRu0="
	sha256HelloWorldB64 = "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek="
	sha512HelloWorldB64 = "MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw=="
)

func ExampleHash() {
	// Example usage of the hash package.
	h, err := hash.New("sha256", sha256HelloWorldB64)
	if err != nil {
		panic(err)
	}

	_, err = h.Write([]byte(helloWorld))
	if err != nil {
		panic(err)
	}

	if !h.Validate() {
		panic("Validate() failed, expected true")
	}

	fmt.Printf("Base64 Digest: %s\n", hash.ToBase64(h.Digest))
	// Output:
	// Base64 Digest: uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=

}

func TestNew(t *testing.T) {
	testCases := []struct {
		algo    string
		digest  string
		wantErr bool
		errStr  string
	}{
		{"sha1", sha1HelloWorldB64, false, ""},
		{"md5", md5HelloWorldB64, false, ""},
		{"sha256", sha256HelloWorldB64, false, ""},
		{"sha512", sha512HelloWorldB64, false, ""},
		{"unsupported", sha512HelloWorldB64, true, "unsupported hash algorithm"},
		{"sha1", "invalid-base64!", true, "invalid base64 digest"},
	}

	for _, tc := range testCases {
		t.Run(tc.algo, func(t *testing.T) {
			_, err := hash.New(tc.algo, tc.digest)
			if (err != nil) != tc.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if err != nil && tc.errStr != "" && !strings.Contains(err.Error(), tc.errStr) {
				t.Errorf("New() error string = %q, want to contain %q", err.Error(), tc.errStr)
			}
		})
	}
}

func TestHash_Validate(t *testing.T) {

	testCases := []struct {
		name    string
		algo    string
		digest  string
		input   string
		isValid bool
	}{
		{"sha1 valid", "sha1", sha1HelloWorldB64, helloWorld, true},
		{"sha1 invalid", "sha1", sha1HelloWorldB64, "goodbye world", false},
		{"md5 valid", "md5", md5HelloWorldB64, helloWorld, true},
		{"md5 invalid", "md5", md5HelloWorldB64, "goodbye world", false},
		{"sha256 valid", "sha256", sha256HelloWorldB64, helloWorld, true},
		{"sha256 invalid", "sha256", sha256HelloWorldB64, "goodbye world", false},
		{"sha512 valid", "sha512", sha512HelloWorldB64, helloWorld, true},
		{"sha512 invalid", "sha512", sha512HelloWorldB64, "goodbye world", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := hash.New(tc.algo, tc.digest)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			_, err = h.Write([]byte(tc.input))
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}

			if got := h.Validate(); got != tc.isValid {
				t.Errorf("%v: Validate() = %v, want %v", tc.algo, got, tc.isValid)
			}

		})
	}
}

func TestBase64Conversion(t *testing.T) {
	original := []byte("some raw binary data")
	encoded := hash.ToBase64(original)
	decoded, err := hash.FromBase64(encoded)
	if err != nil {
		t.Fatalf("FromBase64() failed: %v", err)
	}

	if !bytes.Equal(original, decoded) {
		t.Errorf("roundtrip failed: got %x, want %x", decoded, original)
	}

	_, err = hash.FromBase64("invalid-base64!")
	if err == nil {
		t.Error("FromBase64() should have failed on invalid input, but did not")
	}
}
