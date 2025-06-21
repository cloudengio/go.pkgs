// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package hash provides a simple interface to create and validate hashes
// using various algorithms such as SHA1, MD5, SHA256, and SHA512. The
// hashes are created from base64 encoded digests, which allows for easy
// storage and transmission of hash values.
package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"slices"
)

type Hash struct {
	hash.Hash
	Algo   string
	Digest []byte
}

// New creates a new Hash instance based on the specified algorithm and digest.
// Supported algorithms are "sha1", "md5", "sha256", and "sha512" and the digest
// is base64 encoded.
func New(algo, digest string) (Hash, error) {
	db, err := base64.StdEncoding.DecodeString(digest)
	if err != nil {
		return Hash{}, fmt.Errorf("invalid base64 digest: %w", err)
	}
	// Create the hash instance based on the algorithm.
	h := newHashInstance(algo, db)
	if h.Hash == nil {
		return Hash{}, fmt.Errorf("unsupported hash algorithm: %s", algo)
	}
	h.Algo = algo
	return h, nil
}

func FromBase64(digest string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(digest)
}

func ToBase64(digest []byte) string {
	return base64.StdEncoding.EncodeToString(digest)
}

func newHashInstance(algo string, digest []byte) Hash {
	switch algo {
	case "sha1":
		h := sha1.New()
		return Hash{Hash: h, Digest: []byte(digest)}
	case "md5":
		h := md5.New()
		return Hash{Hash: h, Digest: []byte(digest)}
	case "sha256":
		h := sha256.New()
		return Hash{Hash: h, Digest: []byte(digest)}
	case "sha512":
		h := sha512.New()
		return Hash{Hash: h, Digest: []byte(digest)}
	default:
		return Hash{}
	}
}
func (h Hash) Validate() bool {
	return slices.Equal(h.Sum(nil), h.Digest)
}
