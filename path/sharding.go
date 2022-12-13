// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package path

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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

// SHA1PrefixLength
func SHA1PrefixLength(v int) ShardingOption {
	return func(o *shardingOptions) {
		o.sha1PrefixLen = v
	}
}

// NewSharder returns an instance of Sharder according to the specified
// options. If no options are provided it will behave as if the option
// of SHA1PrefixLength(1) was used.
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

func (s *sha1Sharder) Assign(p string) (prefix, suffix string) {
	sum := sha1.Sum([]byte(p))
	hexsum := hex.EncodeToString(sum[:])
	fmt.Printf(">>> %v\n", hexsum)
	return hexsum[:s.prefix], hexsum[s.prefix:]
}
