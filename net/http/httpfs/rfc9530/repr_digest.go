// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package rfc9530 provides utilities for working with RFC 9530.
// It includes functions for parsing the Repr-Digest header as defined in RFC 9530.
// The Repr-Digest header is used to convey the digest values of representations
// in a format that allows multiple algorithms to be specified.
package rfc9530

import (
	"encoding/base64"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const ReprDigestHeader = "Repr-Digest"

// GenAI: gemini 2.5 wrote this code. Needed several fixes though.

// ParseReprDigest parses the Repr-Digest header value.
// It returns a map of algorithm-to-digest mappings or an error if the format is invalid.
// The digest value is the raw base64 string from the header.
func ParseReprDigest(headerValue string) (map[string]string, error) {
	digests := make(map[string]string)
	parts := strings.SplitSeq(headerValue, ",")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		algo, base64Digest, _, err := ParseAlgoDigest(part)
		if err != nil {
			return nil, err
		}
		digests[algo] = base64Digest
	}

	if len(digests) == 0 {
		return nil, fmt.Errorf("no valid digests found in header value")
	}

	return digests, nil
}

func ParseAlgoDigest(value string) (algo, base64Digest string, bytes []byte, err error) {
	// Split algorithm from the digest value on the first '=' character.
	algo, digest, found := strings.Cut(value, "=")
	if !found {
		return "", "", nil, fmt.Errorf("malformed digest value: %q; missing '=' separator", value)
	}
	algo = strings.TrimSpace(strings.ToLower(algo))
	if algo == "" {
		return "", "", nil, fmt.Errorf("malformed digest value: %q; algorithm is missing", value)
	}
	digest = strings.TrimSpace(digest)
	if len(digest) < 2 {
		// likely missing the the = after the algo, and we've split on =
		// padding of the checksum at the end of the digest value
		return "", "", nil, fmt.Errorf("malformed digest value: %q; digest value is too short", value)
	}
	// The digest must be enclosed in colons
	if digest[0] != ':' || digest[len(digest)-1] != ':' {
		return "", "", nil, fmt.Errorf("malformed digest value for algorithm %q: missing enclosing colons", algo)
	}
	base64Digest = digest[1 : len(digest)-1]

	// Optional: Validate that the digest is valid Base64
	decoded, err := base64.StdEncoding.DecodeString(base64Digest)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid base64 value for algorithm %q: %w", algo, err)
	}

	return algo, base64Digest, decoded, nil
}

// AsHeaderValue formats the algorithm and base64 digest into a Repr-Digest header value.

func AsHeaderValue(algo, base64Digest string) string {
	// Ensure the algorithm is lowercase and the digest is properly formatted
	algo = strings.ToLower(algo)
	return fmt.Sprintf("%s=:%s:", algo, base64Digest)
}

// ChooseDigest selects a digest from the provided map of digests
// based on the specified algorithms with an indication of whether
// the returned digest matches one of the requested algorithms.
// It checks the provided algorithms in order and returns the first
// matching algorithm's digest if found. If no algorithms match,
// it returns the first available digest in the map based on the
// alphabetical order of the keys and a boolean indicating that no
// requested algorithm was matched. The returned value is in the
// format "algo=base64Digest" suitable for the Repr-Digest header.
func ChooseDigest(digests map[string]string, algos ...string) (string, bool) {
	if len(digests) == 0 {
		return "", false // No digests available
	}
	for _, algo := range algos {
		algo = strings.ToLower(algo)
		if digest, ok := digests[algo]; ok {
			return AsHeaderValue(algo, digest), true
		}
	}
	keys := slices.Sorted(maps.Keys(digests))
	return AsHeaderValue(keys[0], digests[keys[0]]), false
}
