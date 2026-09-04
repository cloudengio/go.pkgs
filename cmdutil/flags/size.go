// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"cloudeng.io/cmdutil/cmdtypes"
)

// ByteSize represents a quantity of bytes that can be used as a flag.Value,
// accepting the same notations as cmdtypes.ByteSize: a bare number of bytes,
// or a number with a binary (KiB, MiB, GiB, TiB) or decimal (KB, MB, GB, TB)
// suffix, with an optional space and in any case. It is parsed by
// cmdtypes.ByteSize, so a size given on the command line is read the same way
// as one read from JSON or YAML.
type ByteSize struct {
	opt   string
	value cmdtypes.ByteSize
	set   bool
}

// Set implements flag.Value.
func (b *ByteSize) Set(v string) error {
	size, err := cmdtypes.ParseByteSize(v)
	if err != nil {
		return err
	}
	b.opt = v
	b.value = size
	b.set = true
	return nil
}

// String implements flag.Value. It returns the size as supplied, so that a
// default retains the units it was written in.
func (b *ByteSize) String() string {
	return b.opt
}

// Get implements flag.Getter, returning the number of bytes.
func (b *ByteSize) Get() any {
	return int64(b.value)
}

// Bytes returns the number of bytes represented by the flag.
func (b *ByteSize) Bytes() int64 {
	return int64(b.value)
}

// ByteSize returns the value as a cmdtypes.ByteSize, for use where the same
// value is also read from or written to JSON or YAML.
func (b *ByteSize) ByteSize() cmdtypes.ByteSize {
	return b.value
}

// IsDefault returns true if the value has not been set.
func (b *ByteSize) IsDefault() bool {
	return !b.set
}
