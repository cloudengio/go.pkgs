// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package path //nolint:revive // intentional shadowing

import (
	"crypto/sha1" // #nosec: G505
	"encoding/hex"
)

// Sharder is the interface for assigning and managing pathnames to shards.
type Sharder interface {
	Assign(path string) (prefix, suffix string)
}

type shardingOptions struct {
	sha1PrefixLen int
}

// ShardingOption represents an option to NewPathSharder.
type ShardingOption func(o *shardingOptions)

// WithSHA1PrefixLength requests that a SHA1 sharder with a prefix length
// of v is used. Assigned filenames will be of the form:
// sha1(path)[:v]/sha1(path)[v:]
func WithSHA1PrefixLength(v int) ShardingOption {
	return func(o *shardingOptions) {
		o.sha1PrefixLen = v
	}
}

// NewSharder returns an instance of Sharder according to the specified
// options. If no options are provided it will behave as if the option
// of WithSHA1PrefixLength(2) was used.
func NewSharder(opts ...ShardingOption) Sharder {
	var o shardingOptions
	for _, fn := range opts {
		fn(&o)
	}
	if o.sha1PrefixLen > 0 {
		return &sha1Sharder{o.sha1PrefixLen}
	}
	return &sha1Sharder{1}
}

type sha1Sharder struct {
	prefix int
}

// Assign assigns the supplied path to a shard and returns the
// name (prefix) and filename (suffix) to be used for storing/accessing
// the file.
func (s *sha1Sharder) Assign(p string) (prefix, suffix string) {
	sum := sha1.Sum([]byte(p)) // #nosec G401
	hexsum := hex.EncodeToString(sum[:])
	return hexsum[:s.prefix], hexsum[s.prefix:]
}
