// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import "fmt"

type SizeUnit int64

const (
	Byte SizeUnit = 1

	// base 10
	KB = Byte * 1000
	MB = KB * 1000
	GB = MB * 1000
	TB = GB * 1000
	PB = TB * 1000
	EB = PB * 1000

	// base 2 quantities
	Kib = Byte << 10
	Mib = Kib << 10
	Gib = Mib << 10
	Tib = Gib << 10
	Pib = Tib << 10
	Eib = Pib << 10
)

func (s SizeUnit) String() string {
	switch s {
	case Byte:
		return "B"
	case KB:
		return "KB"
	case Kib:
		return "KiB"
	case MB:
		return "MB"
	case Mib:
		return "MiB"
	case GB:
		return "GB"
	case Gib:
		return "GiB"
	case TB:
		return "TB"
	case Tib:
		return "TiB"
	case PB:
		return "PB"
	case Pib:
		return "PiB"
	case EB:
		return "EB"
	case Eib:
		return "EiB"
	default:
		return ""
	}
}

func (s SizeUnit) Value(v int64) float64 {
	switch s {
	case Byte:
		return float64(v)
	case KB:
		return float64(v) / float64(KB)
	case Kib:
		return float64(v) / float64(Kib)
	case MB:
		return float64(v) / float64(MB)
	case Mib:
		return float64(v) / float64(Mib)
	case GB:
		return float64(v) / float64(GB)
	case Gib:
		return float64(v) / float64(Gib)
	case TB:
		return float64(v) / float64(TB)
	case Tib:
		return float64(v) / float64(Tib)
	case PB:
		return float64(v) / float64(PB)
	case Pib:
		return float64(v) / float64(Pib)
	case EB:
		return float64(v) / float64(EB)
	case Eib:
		return float64(v) / float64(Eib)
	default:
		return 0.0
	}
}

func DecimalUnitForSize(size int64) SizeUnit {
	if size < 0 {
		return Byte
	} else if size < int64(KB) {
		return Byte
	} else if size < int64(MB) {
		return KB
	} else if size < int64(GB) {
		return MB
	} else if size < int64(TB) {
		return GB
	} else if size < int64(PB) {
		return TB
	} else if size < int64(EB) {
		return PB
	}
	return EB
}

func BinaryUnitForSize(size int64) SizeUnit {
	if size < 0 {
		return Byte
	} else if size < int64(Kib) {
		return Byte
	} else if size < int64(Mib) {
		return Kib
	} else if size < int64(Gib) {
		return Mib
	} else if size < int64(Tib) {
		return Gib
	} else if size < int64(Pib) {
		return Tib
	} else if size < int64(Eib) {
		return Pib
	}
	return Eib
}

func BinarySize(width, precision int, val int64) string {
	unit := BinaryUnitForSize(val)
	return fmt.Sprintf("%[1]*.[2]*[3]f%v", width, precision, unit.Value(val), unit.String())
}

func DecimalSize(width, precision int, val int64) string {
	unit := DecimalUnitForSize(val)
	fmt.Printf("unit: %v, value: %d => %v\n", unit, val, unit.Value(val))
	return fmt.Sprintf("%[1]*.[2]*[3]f%v", width, precision, unit.Value(val), unit.String())
}
