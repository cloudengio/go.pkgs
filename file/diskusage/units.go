// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage

import (
	"fmt"
	"strconv"
	"strings"
)

// Binary represents a number of bytes in base 2.
type Binary SizeUnit

func (b Binary) Value(value int64) float64 {
	return float64(value) / float64(b)
}

func (b Binary) Standardize() (float64, string) {
	u := BinaryUnitForSize(int64(b))
	return u.Value(int64(b)), u.String()
}

func (b Binary) Format(f fmt.State, verb rune) {
	SizeUnit(b).format(f, verb, true)
}

// Decimal represents a number of bytes in base 10.
type Decimal SizeUnit

func (b Decimal) Value(value int64) float64 {
	return float64(value) / float64(b)
}

func (b Decimal) Standardize() (float64, string) {
	u := DecimalUnitForSize(int64(b))
	return u.Value(int64(b)), u.String()
}

func (b Decimal) Format(f fmt.State, verb rune) {
	SizeUnit(b).format(f, verb, false)
}

type suffixSpec struct {
	units string
	scale int64
}

var (
	decimalSuffixes = []suffixSpec{
		{"EB", int64(EB)},
		{"PB", int64(PB)},
		{"TB", int64(TB)},
		{"GB", int64(GB)},
		{"MB", int64(MB)},
		{"KB", int64(KB)},
	}

	binarySuffixes = []suffixSpec{
		{"EiB", int64(EiB)},
		{"PiB", int64(PiB)},
		{"TiB", int64(TiB)},
		{"GiB", int64(GiB)},
		{"MiB", int64(MiB)},
		{"KiB", int64(KiB)},
	}
)

func searchSuffixes(suffix string, suffixes []suffixSpec) (int64, bool) {
	for _, s := range suffixes {
		if s.units == suffix {
			return s.scale, true
		}
	}
	return 0, false
}

func handleSuffix(val string, sl int, suffixes []suffixSpec) (float64, bool) {
	suffix := val[len(val)-sl:]
	if s, ok := searchSuffixes(suffix, suffixes); ok {
		v, err := strconv.ParseFloat(val[:len(val)-sl], 64)
		if err != nil {
			return 0, false
		}
		return v * float64(s), true
	}
	return 0, false
}

func ParseToBytes(val string) (float64, error) {
	if len(val) == 0 {
		return 0, nil
	}
	val = strings.ReplaceAll(val, ",", "")
	if len(val) <= 2 || val[len(val)-1] != 'B' {
		return strconv.ParseFloat(val, 64)
	}
	if s, ok := handleSuffix(val, 2, decimalSuffixes); ok {
		return s, nil
	}
	if s, ok := handleSuffix(val, 3, binarySuffixes); ok {
		return s, nil
	}
	return strconv.ParseFloat(val, 64)
}
