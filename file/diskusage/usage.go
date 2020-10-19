// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage

import (
	"fmt"
	"strconv"
)

type Calculator interface {
	Calculate(int64) int64
	String() string
}

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

func (r0 RAID0) Calculate(size int64) int64 {
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

type Simple struct {
	blockSize   int64
	description string
}

func NewSimple(blocksize int64) Calculator {
	return Simple{
		blockSize:   blocksize,
		description: "simple: " + strconv.FormatInt(blocksize, 10),
	}
}

func (s Simple) Calculate(size int64) int64 {
	return ((size + s.blockSize) / s.blockSize) * s.blockSize
}

func (s Simple) String() string {
	return s.description
}

type Identity struct{}

func NewIdentity() Calculator {
	return &Identity{}
}

func (i Identity) Calculate(size int64) int64 {
	return size
}

func (i Identity) String() string {
	return "identity"
}
