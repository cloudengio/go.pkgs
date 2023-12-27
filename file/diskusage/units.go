// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage

import (
	"strconv"
	"strings"
)

// Base2Bytes represents a number of bytes in base 2.
type Base2Bytes int64

// Values for Base2Bytes.
const (
	KiB Base2Bytes = 1024
	MiB Base2Bytes = KiB * 1024
	GiB Base2Bytes = MiB * 1024
	TiB Base2Bytes = GiB * 1024
	PiB Base2Bytes = TiB * 1024
	EiB Base2Bytes = PiB * 1024
)

// Base2Bytes represents a number of bytes in base 10.
type DecimalBytes int64

// Values for DecimalBytes.
const (
	KB DecimalBytes = 1000
	MB DecimalBytes = KB * 1000
	GB DecimalBytes = MB * 1000
	TB DecimalBytes = GB * 1000
	PB DecimalBytes = TB * 1000
	EB DecimalBytes = PB * 1000
)

func (b Base2Bytes) Num(value int64) float64 {
	return float64(value) / float64(b)
}

func (b Base2Bytes) Standardize() (float64, string) {
	v := int64(b)
	switch {
	case b > EiB:
		return EiB.Num(v), "EiB"
	case b >= PiB:
		return PiB.Num(v), "PiB"
	case b >= TiB:
		return TiB.Num(v), "TiB"
	case b >= GiB:
		return GiB.Num(v), "GiB"
	case b >= MiB:
		return MiB.Num(v), "MiB"
	default:
		return KiB.Num(v), "KiB"
	}
}

func (b DecimalBytes) Num(value int64) float64 {
	return float64(value) / float64(b)
}

func (b DecimalBytes) Standardize() (float64, string) {
	v := int64(b)
	switch {
	case b >= EB:
		return EB.Num(v), "EB"
	case b >= PB:
		return PB.Num(v), "PB"
	case b >= TB:
		return TB.Num(v), "TB"
	case b >= GB:
		return GB.Num(v), "GB"
	case b >= MB:
		return MB.Num(v), "MB"
	default:
		return KB.Num(v), "KB"
	}
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
