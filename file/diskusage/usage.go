// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage

import (
	"fmt"
	"strconv"
)

// Calculator is used to calculate the size of a file or directory
// based on either its size in bytes (often referred to as its apparent size)
// and/or the number of storage blocks it occupies. Some file systems
// support sparse files (most unix filesystems) where the number of blocks
// occupied by a file is less than the number of bytes it represents, hence,
// the term 'apparent size'.
type Calculator interface {
	Calculate(bytes, blocks int64) int64
	String() string
}

// RAID0 is a calculator for RAID0 volumes based on the apparent size
// of the file and the RAID0 stripe size and number of stripes.
type RAID0 struct {
	stripeSize  int64
	numStripes  int
	description string
}

func NewRAID0(stripeSize int64, numStripes int) Calculator {
	return RAID0{
		stripeSize:  stripeSize,
		numStripes:  numStripes,
		description: fmt.Sprintf("raid0: %v/%v", numStripes, stripeSize),
	}
}

func (r0 RAID0) Calculate(size, blocks int64) int64 {
	raw := ((size + r0.stripeSize) / r0.stripeSize) * r0.stripeSize
	striped := int64(r0.numStripes) * r0.stripeSize
	if striped > raw {
		return striped
	}
	return raw
}

func (r0 RAID0) String() string {
	return r0.description
}

// Roundup rounds up the apparent size of a file to the nearest
// block size multiple.
type Roundup struct {
	blockSize   int64
	description string
}

func NewRoundup(blocksize int64) Calculator {
	return Roundup{
		blockSize:   blocksize,
		description: "roundup: " + strconv.FormatInt(blocksize, 10),
	}
}

func (s Roundup) Calculate(bytes, blocks int64) int64 {
	return ((bytes + s.blockSize) / s.blockSize) * s.blockSize
}

func (s Roundup) String() string {
	return s.description
}

type Block struct {
	blockSize   int64
	description string
}

// Block uses the number of blocks occupied by a file to calculate its size.
func NewBlock(blocksize int64) Calculator {
	return Block{
		blockSize:   blocksize,
		description: "block: " + strconv.FormatInt(blocksize, 10),
	}
}

func (s Block) Calculate(bytes, blocks int64) int64 {
	return blocks * s.blockSize
}

func (s Block) String() string {
	return s.description
}

// Identity returns the apparent size of a file.
type Identity struct{}

func NewIdentity() Calculator {
	return &Identity{}
}

func (i Identity) Calculate(bytes, blocks int64) int64 {
	return bytes
}

func (i Identity) String() string {
	return "identity"
}
