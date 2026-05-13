// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ByteSize represents a quantity of bytes. It can be parsed from and marshaled
// to human-readable strings using either binary (KiB, MiB, GiB, TiB) or
// decimal (KB, MB, GB, TB) unit suffixes. A space between the number and unit
// is optional; parsing is case-insensitive. Bare integers are treated as bytes.
// Floating-point values are accepted during parsing (e.g. "1.5GiB").
type ByteSize int64

const (
	Byte ByteSize = 1

	KB ByteSize = 1_000
	MB          = 1_000 * KB
	GB          = 1_000 * MB
	TB          = 1_000 * GB

	KiB ByteSize = 1_024
	MiB          = 1_024 * KiB
	GiB          = 1_024 * MiB
	TiB          = 1_024 * GiB
)

// Decimal units are checked before binary units so that values produced by
// decimal constants (e.g. 1TB) are not displayed as a binary multiple (e.g.
// 976562500KiB) in String(). Pure binary values such as 1GiB are not evenly
// divisible by any decimal unit, so they fall through to the binary entries.
var sizeUnits = []struct {
	v ByteSize
	s string
}{
	{TB, "TB"}, {GB, "GB"}, {MB, "MB"}, {KB, "KB"},
	{TiB, "TiB"}, {GiB, "GiB"}, {MiB, "MiB"}, {KiB, "KiB"},
}

// ParseByteSize parses s into a ByteSize. Binary (KiB, MiB, GiB, TiB) and
// decimal (KB, MB, GB, TB) suffixes are supported. A space between the number
// and unit is allowed; parsing is case-insensitive. A bare number is treated
// as bytes. Floating-point values are rounded to the nearest byte.
func ParseByteSize(s string) (ByteSize, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty byte size string")
	}

	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
		i++
	}

	numStr := strings.TrimSpace(s[:i])
	unitStr := strings.TrimSpace(s[i:])

	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid byte size %q: %w", s, err)
	}

	multiplier, err := getMultiplier(unitStr, s)
	if err != nil {
		return 0, err
	}

	result := math.Round(f * multiplier)
	// 1<<63 is exactly representable as float64 (it is 2^63).
	// Valid int64 range is [-2^63, 2^63-1], so reject result >= 2^63 or < -2^63.
	const (
		maxF = float64(1 << 63)
		minF = -float64(1 << 63)
	)
	if result >= maxF || result < minF {
		return 0, fmt.Errorf("byte size %q overflows int64", s)
	}
	return ByteSize(result), nil
}

func getMultiplier(unitStr, s string) (float64, error) {
	var multiplier float64
	switch strings.ToUpper(unitStr) {
	case "", "B":
		multiplier = 1
	case "KB":
		multiplier = float64(KB)
	case "MB":
		multiplier = float64(MB)
	case "GB":
		multiplier = float64(GB)
	case "TB":
		multiplier = float64(TB)
	case "KIB":
		multiplier = float64(KiB)
	case "MIB":
		multiplier = float64(MiB)
	case "GIB":
		multiplier = float64(GiB)
	case "TIB":
		multiplier = float64(TiB)
	default:
		return 0, fmt.Errorf("unknown unit %q in byte size %q", unitStr, s)
	}
	return multiplier, nil
}

// String returns a human-readable representation of b. It selects the largest
// binary unit (TiB, GiB, MiB, KiB) that divides b evenly, then the largest
// decimal unit (TB, GB, MB, KB), and falls back to "NB" when no unit divides
// evenly.
func (b ByteSize) String() string {
	if b == 0 {
		return "0B"
	}
	neg := b < 0
	uabs := uint64(b)
	if neg {
		uabs = -uint64(b)
	}
	for _, u := range sizeUnits {
		if uabs%uint64(u.v) == 0 {
			n := uabs / uint64(u.v)
			if neg {
				return fmt.Sprintf("-%d%s", n, u.s)
			}
			return fmt.Sprintf("%d%s", n, u.s)
		}
	}
	if neg {
		return fmt.Sprintf("-%dB", uabs)
	}
	return fmt.Sprintf("%dB", uabs)
}

func (b ByteSize) MarshalYAML() (any, error) {
	return b.String(), nil
}

func (b *ByteSize) UnmarshalYAML(value *yaml.Node) error {
	v, err := ParseByteSize(value.Value)
	if err != nil {
		return err
	}
	*b = v
	return nil
}
