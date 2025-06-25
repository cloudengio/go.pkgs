// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package digests_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/algo/digests"
)

const (
	helloWorld = "hello world"

	// hex encoded digests for "hello world"
	md5HelloWorldHex    = "5eb63bbbe01eeed093cb22bb8f5acdc3"
	sha1HelloWorldHex   = "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"
	sha256HelloWorldHex = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	sha512HelloWorldHex = "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f"

	// base64 encoded digests for "hello world"
	md5HelloWorldB64    = "XrY7u+Ae7tCTyyK7j1rNww=="
	sha1HelloWorldB64   = "Kq5sNclPz7QV2+lfQIuc6R7oRu0="
	sha256HelloWorldB64 = "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek="
	sha512HelloWorldB64 = "MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw=="
)

func TestNew(t *testing.T) {
	testCases := []struct {
		algo    string
		wantErr bool
		errStr  string
	}{
		{"sha1", false, ""},
		{"md5", false, ""},
		{"sha256", false, ""},
		{"sha512", false, ""},
		{"unsupported", true, "unsupported hash algorithm: unsupported"},
	}

	for _, tc := range testCases {
		t.Run(tc.algo, func(t *testing.T) {
			_, err := digests.New(tc.algo, []byte("digest"))
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

func TestConverters(t *testing.T) {
	t.Run("base64", func(t *testing.T) {
		original := []byte("some raw binary data")
		encoded := digests.ToBase64(original)
		decoded, err := digests.FromBase64(encoded)
		if err != nil {
			t.Fatalf("FromBase64() failed: %v", err)
		}
		if !bytes.Equal(original, decoded) {
			t.Errorf("base64 roundtrip failed: got %x, want %x", decoded, original)
		}
		_, err = digests.FromBase64("!@#$")
		if err == nil {
			t.Error("FromBase64() should have failed on invalid input, but did not")
		}
	})

	t.Run("hex", func(t *testing.T) {
		original := []byte("some raw binary data")
		encoded := digests.ToHex(original)
		decoded, err := digests.FromHex(encoded)
		if err != nil {
			t.Fatalf("FromHex() failed: %v", err)
		}
		if !bytes.Equal(original, decoded) {
			t.Errorf("hex roundtrip failed: got %x, want %x", decoded, original)
		}
		_, err = digests.FromHex("not-valid-hex")
		if err == nil {
			t.Error("FromHex() should have failed on invalid input, but did not")
		}
	})

}

func TestConversions(t *testing.T) {
	for _, tc := range []struct {
		name string
		b64  string
		hex  string
	}{
		{"sha1", sha1HelloWorldB64, sha1HelloWorldHex},
		{"md5", md5HelloWorldB64, md5HelloWorldHex},
		{"sha256", sha256HelloWorldB64, sha256HelloWorldHex},
		{"sha512", sha512HelloWorldB64, sha512HelloWorldHex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Test Base64 to Hex conversion
			decoded, err := digests.FromBase64(tc.b64)
			if err != nil {
				t.Fatalf("FromBase64() failed: %v", err)
			}
			hex := digests.ToHex(decoded)
			if hex != tc.hex {
				t.Errorf("ToHex() = %s, want %s", hex, tc.hex)
			}

			// Test Hex to Base64 conversion
			decoded, err = digests.FromHex(tc.hex)
			if err != nil {
				t.Fatalf("FromHex() failed: %v", err)
			}
			b64 := digests.ToBase64(decoded)
			if b64 != tc.b64 {
				t.Errorf("ToBase64() = %s, want %s", b64, tc.b64)
			}
		})
	}
}

func TestHash_ValidateHex(t *testing.T) {
	testCases := []struct {
		name      string
		algo      string
		hexDigest string
		input     string
		isValid   bool
	}{
		{"sha1 valid", "sha1", sha1HelloWorldHex, helloWorld, true},
		{"sha1 invalid", "sha1", sha1HelloWorldHex, "goodbye world", false},
		{"md5 valid", "md5", md5HelloWorldHex, helloWorld, true},
		{"md5 invalid", "md5", md5HelloWorldHex, "goodbye world", false},
		{"sha256 valid", "sha256", sha256HelloWorldHex, helloWorld, true},
		{"sha256 invalid", "sha256", sha256HelloWorldHex, "goodbye world", false},
		{"sha512 valid", "sha512", sha512HelloWorldHex, helloWorld, true},
		{"sha512 invalid", "sha512", sha512HelloWorldHex, "goodbye world", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			digest, err := digests.FromHex(tc.hexDigest)
			if err != nil {
				t.Fatalf("FromHex() failed: %v", err)
			}

			h, err := digests.New(tc.algo, digest)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			_, err = h.Write([]byte(tc.input))
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}

			if got := h.Validate(); got != tc.isValid {
				t.Errorf("Validate() = %v, want %v", got, tc.isValid)
			}
		})
	}
}

func TestHash_ValidateBase64(t *testing.T) {
	testCases := []struct {
		name      string
		algo      string
		hexDigest string
		input     string
		isValid   bool
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
			digest, err := digests.FromBase64(tc.hexDigest)
			if err != nil {
				t.Fatalf("FromBase64() failed: %v", err)
			}

			h, err := digests.New(tc.algo, digest)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			_, err = h.Write([]byte(tc.input))
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}

			if got := h.Validate(); got != tc.isValid {
				t.Errorf("Validate() = %v, want %v", got, tc.isValid)
			}
		})
	}
}

func TestParseHex(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantAlgo   string
		wantHex    string
		wantErr    bool
		wantErrStr string
	}{
		{"valid sha256", "sha256=aabbcc", "sha256", "aabbcc", false, ""},
		{"valid md5", "md5=1234567890abcdef", "md5", "1234567890abcdef", false, ""},
		{"empty digest", "sha1=", "sha1", "", false, ""},
		{"missing separator", "sha256aabbcc", "", "", true, "failed to parse hex digest"},
		{"invalid hex", "sha256=not-hex", "", "", true, "invalid hex digest"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotAlgo, gotHex, err := digests.ParseHex(tc.input)

			if (err != nil) != tc.wantErr {
				t.Errorf("ParseHex() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if err != nil && tc.wantErrStr != "" && !strings.Contains(err.Error(), tc.wantErrStr) {
				t.Errorf("ParseHex() error string = %q, want to contain %q", err.Error(), tc.wantErrStr)
			}

			if gotAlgo != tc.wantAlgo {
				t.Errorf("ParseHex() gotAlgo = %v, want %v", gotAlgo, tc.wantAlgo)
			}
			if !reflect.DeepEqual(gotHex, tc.wantHex) {
				t.Errorf("ParseHex() gotHex = %v, want %v", gotHex, tc.wantHex)
			}
		})
	}
}
