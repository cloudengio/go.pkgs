// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package digests provides a simple interface to create and validate digests
// using various algorithms such as SHA1, MD5, SHA256, and SHA512. Support
// is provided for working with digests in both base64 and hex formats.
package digests

import (
	"crypto/md5"  //nolint:gosec // G401: Use of weak cryptographic primitive
	"crypto/sha1" //nolint:gosec // G401: Use of weak cryptographic primitive
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"slices"
	"strings"
)

type Hash struct {
	hash.Hash
	Algo   string
	Digest []byte
}

const (
	MD5    = "md5"
	SHA1   = "sha1"
	SHA256 = "sha256"
	SHA512 = "sha512"
)

// New creates a new Hash instance based on the specified algorithm.
// Currently supported algorithms are "sha1", "md5", "sha256", and "sha512".
//
// Note: MD5 and SHA1 are cryptographically weak and should not be used for
// security-sensitive applications.
func New(algo string, digest []byte) (Hash, error) {
	h := newHashInstance(algo)
	if h == nil {
		return Hash{}, fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
	return Hash{Hash: h, Algo: algo, Digest: digest}, nil
}

func FromBase64(digest string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(digest)
}

func ToBase64(digest []byte) string {
	return base64.StdEncoding.EncodeToString(digest)
}

func FromHex(digest string) ([]byte, error) {
	return hex.DecodeString(digest)
}

func ToHex(digest []byte) string {
	return hex.EncodeToString(digest)
}

func (h Hash) IsSet() bool {
	return h.Hash != nil && len(h.Digest) > 0
}

func IsSupported(algo string) bool {
	switch algo {
	case "sha1", "md5", "sha256", "sha512":
		return true
	case "sha-1", "sha-256", "sha-512":
		// Allow for hyphenated versions for compatibility with some headers.
		return true
	default:
		return false
	}
}

// Supported returns a list of supported hash algorithms, note that hyphenated
// versions of SHA1, SHA256, and SHA512 are also included for compatibility,
// e.g., "sha-1", "sha-256", "sha-512" as well as the non-hyphenated
// versions "sha1", "sha256", and "sha512".
func Supported() []string {
	return []string{"sha1", "md5", "sha256", "sha512", "sha-1", "sha-256", "sha-512"}
}

func newHashInstance(algo string) hash.Hash {
	switch algo {
	case "sha1", "sha-1":
		return sha1.New() //nolint:gosec // G401: Use of weak cryptographic primitive
	case "md5":
		return md5.New() //nolint:gosec // G401: Use of weak cryptographic primitive
	case "sha256", "sha-256":
		return sha256.New()
	case "sha512", "sha-512":
		return sha512.New()
	default:
		return nil
	}
}

// Validate checks if the hash instance's computed sum matches the expected digest.
func (h Hash) Validate() bool {
	return h.IsSet() && slices.Equal(h.Sum(nil), h.Digest)
}

// ParseHex decodes a digest specification of the form <algo>=<hex-digits>.
func ParseHex(digest string) (algo, hexdigits string, err error) {
	algo, digest, ok := strings.Cut(digest, "=")
	if !ok {
		return "", "", fmt.Errorf("failed to parse hex digest of form <algo>=<hex-digits>: %q", digest)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", "", fmt.Errorf("invalid hex digest: %w", err)
	}
	return algo, digest, nil
}
