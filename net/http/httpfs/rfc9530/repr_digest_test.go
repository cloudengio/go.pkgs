// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package rfc9530_test

import (
	"encoding/base64"
	"reflect"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/net/http/httpfs/rfc9530"
)

// GenAI: gemini 2.5 wrote this code. Some of the tests needed to be corrected.

const (
	// SHA-256 hash of an empty string (""), base64 encoded
	sha256EmptyStringB64 = "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="
	// SHA-512 hash of an empty string (""), base64 encoded
	sha512EmptyStringB64 = "z4PhNX7vuL3xVChQ1m2AB9Yg5AULVxXcg/SpIdNs6c5H0NE8XYXysP+DGNKHfuwvY7kxvUdBeoGlODJ6+SfaPg=="
	// SHA-256 hash of "any carnal pleasure", base64 encoded
	sha256AnyCarnalPleasureB64 = "JVqCgJGHcJ8Hk/5qf2gACBDQBVZ48H9QIHFXFQeRIAI="
	// id-sha-256 (alternative name for sha-256)
	idSha256ValueB64 = "ALGOIDTESTVALUEFORIDSHA256=="
	// md5 (example of another algorithm)
	md5ValueB64 = "1B2M2Y8AsgTpgAmY7PhCfg==" // md5 of "hello"
)

func TestParseReprDigest(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		want        map[string]string
		wantErr     string // Substring of the expected error
	}{
		{
			name:        "valid single digest sha-256",
			headerValue: "sha-256=:" + sha256EmptyStringB64 + ":",
			want:        map[string]string{"sha-256": sha256EmptyStringB64},
		},
		{
			name:        "valid single digest sha-512",
			headerValue: "sha-512=:" + sha512EmptyStringB64 + ":",
			want:        map[string]string{"sha-512": sha512EmptyStringB64},
		},
		{
			name:        "valid multiple digests",
			headerValue: "sha-256=:" + sha256EmptyStringB64 + ":, sha-512=:" + sha512EmptyStringB64 + ":",
			want:        map[string]string{"sha-256": sha256EmptyStringB64, "sha-512": sha512EmptyStringB64},
		},
		{
			name:        "valid multiple digests with extra spaces",
			headerValue: "  sha-256 = :" + sha256EmptyStringB64 + ": ,  sha-512 = :" + sha512EmptyStringB64 + ":  ",
			want:        map[string]string{"sha-256": sha256EmptyStringB64, "sha-512": sha512EmptyStringB64},
		},
		{
			name:        "algorithm with different casing",
			headerValue: "SHA-256=:" + sha256EmptyStringB64 + ":",
			want:        map[string]string{"sha-256": sha256EmptyStringB64},
		},
		{
			name:        "empty digest value (valid base64 for empty string)",
			headerValue: "sha-256=::", // An empty base64 string is valid for an empty byte sequence
			want:        map[string]string{"sha-256": ""},
		},
		{
			name:        "digest with padding (represents actual hash)",
			headerValue: "sha-256=:" + sha256AnyCarnalPleasureB64 + ":",
			want:        map[string]string{"sha-256": sha256AnyCarnalPleasureB64},
		},
		{
			name:        "custom algorithm name with realistic-looking value",
			headerValue: "my-custom-algo=:" + sha256EmptyStringB64 + ":", // Using a known valid hash for structure
			want:        map[string]string{"my-custom-algo": sha256EmptyStringB64},
		},
		{
			name:        "empty header value",
			headerValue: "",
			wantErr:     "no valid digests found in header value",
		},
		{
			name:        "header value with only spaces",
			headerValue: "   ",
			wantErr:     "no valid digests found in header value",
		},
		{
			name:        "header value with only commas",
			headerValue: ",,,",
			wantErr:     "no valid digests found in header value",
		},
		{
			name:        "malformed entry - missing equals",
			headerValue: "sha-256:" + sha256EmptyStringB64 + ":",
			wantErr:     "digest value is too short",
		},
		{
			name:        "malformed entry - missing algorithm",
			headerValue: "=:" + sha256EmptyStringB64 + ":",
			wantErr:     "algorithm is missing",
		},
		{
			name:        "malformed entry - missing leading colon",
			headerValue: "sha-256=" + sha256EmptyStringB64 + "::",
			wantErr:     "missing enclosing colons",
		},
		{
			name:        "malformed entry - missing trailing colon",
			headerValue: "sha-256=:" + sha256EmptyStringB64,
			wantErr:     "missing enclosing colons",
		},
		{
			name:        "malformed entry - no colons",
			headerValue: "sha-256=" + sha256EmptyStringB64,
			wantErr:     "missing enclosing colons",
		},
		{
			name:        "invalid base64 value",
			headerValue: "sha-256=:Invalid Base64*:, sha-512=:" + sha512EmptyStringB64 + ":",
			wantErr:     "invalid base64 value for algorithm \"sha-256\"",
		},
		{
			name:        "mixed valid and invalid base64 (should fail on first invalid)",
			headerValue: "sha-512=:" + sha512EmptyStringB64 + ":, sha-256=:Invalid Base64*:",
			wantErr:     "invalid base64 value for algorithm \"sha-256\"",
		},
		{
			name:        "malformed entry among valid ones - missing equals",
			headerValue: "sha-256=:" + sha256EmptyStringB64 + ":, sha-512:" + sha512EmptyStringB64 + ":",
			wantErr:     "missing enclosing colons",
		},
		{
			name:        "malformed entry among valid ones - missing colons",
			headerValue: "sha-256=:" + sha256EmptyStringB64 + ":, sha-512=" + sha512EmptyStringB64,
			wantErr:     "missing enclosing colons",
		},
		{
			name:        "trailing comma",
			headerValue: "sha-256=:" + sha256EmptyStringB64 + ":,",
			want:        map[string]string{"sha-256": sha256EmptyStringB64},
		},
		{
			name:        "leading comma",
			headerValue: ",sha-256=:" + sha256EmptyStringB64 + ":",
			want:        map[string]string{"sha-256": sha256EmptyStringB64},
		},
		{
			name:        "multiple commas between entries",
			headerValue: "sha-256=:" + sha256EmptyStringB64 + ":, ,, sha-512=:" + sha512EmptyStringB64 + ":",
			want:        map[string]string{"sha-256": sha256EmptyStringB64, "sha-512": sha512EmptyStringB64},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rfc9530.ParseReprDigest(tt.headerValue)

			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("%v: ParseReprDigest() error = nil, wantErr %q", tt.name, tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("%v: ParseReprDigest() error = %q, wantErr %q", tt.name, err.Error(), tt.wantErr)
				}
				return // Expected error, no need to check 'want'
			}

			if err != nil {
				t.Fatalf("%v: ParseReprDigest() unexpected error = %v", tt.name, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%v: ParseReprDigest() got = %v, want %v", tt.name, got, tt.want)
			}
			for k, v := range got {
				d := k + "=:" + v + ":"
				algo, b64, raw, err := rfc9530.ParseAlgoDigest(d)
				if err != nil {
					t.Errorf("%v: ParseAlgoDigest() hdr %q error = %v", tt.name, d, err)
					return
				}
				if algo == "" {
					t.Errorf("%v: ParseAlgoDigest() algo = empty, want non-empty", tt.name)
				}
				if b64 != v {
					t.Errorf("%v: ParseAlgoDigest() b64 = %q, want %q", tt.name, b64, v)
				}
				expectedRaw, err := base64.StdEncoding.DecodeString(v)
				if err != nil {
					t.Fatalf("invalid base64 in test case %q: %v", v, err)
				}
				if !slices.Equal(raw, expectedRaw) {
					t.Errorf("%v: ParseAlgoDigest() raw bytes mismatch: got %x, want %x", tt.name, raw, expectedRaw)
				}
			}
		})
	}
}

func TestChooseDigest(t *testing.T) {
	availableDigests := map[string]string{
		"sha-512":    sha512EmptyStringB64,
		"sha-256":    sha256EmptyStringB64,
		"id-sha-256": idSha256ValueB64,
		"md5":        md5ValueB64,
	}

	tests := []struct {
		name           string
		digests        map[string]string
		preferredAlgos []string
		wantHeaderVal  string
		wantMatched    bool
	}{
		{
			name:           "first preferred algo (sha-256) exists",
			digests:        availableDigests,
			preferredAlgos: []string{"sha-256", "sha-512"},
			wantHeaderVal:  "sha-256=:" + sha256EmptyStringB64 + ":",
			wantMatched:    true,
		},
		{
			name:           "second preferred algo (sha-512) exists",
			digests:        availableDigests,
			preferredAlgos: []string{"non-existent", "sha-512"},
			wantHeaderVal:  "sha-512=:" + sha512EmptyStringB64 + ":",
			wantMatched:    true,
		},
		{
			name:           "preferred algo with different casing (SHA-256)",
			digests:        availableDigests,
			preferredAlgos: []string{"SHA-256"},
			wantHeaderVal:  "sha-256=:" + sha256EmptyStringB64 + ":",
			wantMatched:    true,
		},
		{
			name:           "id-sha-256 preferred",
			digests:        availableDigests,
			preferredAlgos: []string{"id-sha-256"},
			wantHeaderVal:  "id-sha-256=:" + idSha256ValueB64 + ":",
			wantMatched:    true,
		},
		{
			name:           "no preferred algo exists, fallback to first sorted (id-sha-256)",
			digests:        availableDigests,
			preferredAlgos: []string{"non-existent-1", "non-existent-2"},
			wantHeaderVal:  "id-sha-256=:" + idSha256ValueB64 + ":", // 'id-sha-256' is first alphabetically
			wantMatched:    false,
		},
		{
			name:           "empty preferred algos list, fallback to first sorted (id-sha-256)",
			digests:        availableDigests,
			preferredAlgos: []string{},
			wantHeaderVal:  "id-sha-256=:" + idSha256ValueB64 + ":",
			wantMatched:    false,
		},
		{
			name:           "nil preferred algos list, fallback to first sorted (id-sha-256)",
			digests:        availableDigests,
			preferredAlgos: nil,
			wantHeaderVal:  "id-sha-256=:" + idSha256ValueB64 + ":",
			wantMatched:    false,
		},
		{
			name:           "empty digests map",
			digests:        map[string]string{},
			preferredAlgos: []string{"sha-256"},
			wantHeaderVal:  "", // Expect empty string as per current implementation for empty map
			wantMatched:    false,
		},
		{
			name:           "nil digests map",
			digests:        nil,
			preferredAlgos: []string{"sha-256"},
			wantHeaderVal:  "", // Expect empty string as per current implementation for nil map
			wantMatched:    false,
		},
		{
			name:           "md5 preferred and available",
			digests:        availableDigests,
			preferredAlgos: []string{"md5"},
			wantHeaderVal:  "md5=:" + md5ValueB64 + ":",
			wantMatched:    true,
		},
		{
			name:           "preference order respected (sha-512 before sha-256)",
			digests:        availableDigests,
			preferredAlgos: []string{"sha-512", "sha-256"},
			wantHeaderVal:  "sha-512=:" + sha512EmptyStringB64 + ":",
			wantMatched:    true,
		},
		{
			name: "only one digest available, no preference match",
			digests: map[string]string{
				"sha-256": sha256EmptyStringB64,
			},
			preferredAlgos: []string{"sha-512"},
			wantHeaderVal:  "sha-256=:" + sha256EmptyStringB64 + ":",
			wantMatched:    false,
		},
		{
			name: "only one digest available, preference match",
			digests: map[string]string{
				"sha-256": sha256EmptyStringB64,
			},
			preferredAlgos: []string{"sha-256"},
			wantHeaderVal:  "sha-256=:" + sha256EmptyStringB64 + ":",
			wantMatched:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headerVal, matched := rfc9530.ChooseDigest(tt.digests, tt.preferredAlgos...)

			if headerVal != tt.wantHeaderVal {
				t.Errorf("ChooseDigest() headerVal = %q, want %q", headerVal, tt.wantHeaderVal)
			}
			if matched != tt.wantMatched {
				t.Errorf("ChooseDigest() matched = %v, want %v", matched, tt.wantMatched)
			}
		})
	}
}
