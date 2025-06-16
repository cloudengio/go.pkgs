// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage

import "fmt"

// SizeUnit represents a unit of size in bytes. It can be used to represent
// both decimal and binary sizes.
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
	KiB = Byte << 10
	MiB = KiB << 10
	GiB = MiB << 10
	TiB = GiB << 10
	PiB = TiB << 10
	EiB = PiB << 10
)

func (s SizeUnit) String() string {
	switch s {
	case Byte:
		return "B"
	case KB:
		return "KB"
	case KiB:
		return "KiB"
	case MB:
		return "MB"
	case MiB:
		return "MiB"
	case GB:
		return "GB"
	case GiB:
		return "GiB"
	case TB:
		return "TB"
	case TiB:
		return "TiB"
	case PB:
		return "PB"
	case PiB:
		return "PiB"
	case EB:
		return "EB"
	case EiB:
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
	case KiB:
		return float64(v) / float64(KiB)
	case MB:
		return float64(v) / float64(MB)
	case MiB:
		return float64(v) / float64(MiB)
	case GB:
		return float64(v) / float64(GB)
	case GiB:
		return float64(v) / float64(GiB)
	case TB:
		return float64(v) / float64(TB)
	case TiB:
		return float64(v) / float64(TiB)
	case PB:
		return float64(v) / float64(PB)
	case PiB:
		return float64(v) / float64(PiB)
	case EB:
		return float64(v) / float64(EB)
	case EiB:
		return float64(v) / float64(EiB)
	default:
		return 0.0
	}
}

func DecimalUnitForSize(size int64) SizeUnit {
	switch {
	case size < int64(KB):
		return Byte
	case size < int64(MB):
		return KB
	case size < int64(GB):
		return MB
	case size < int64(TB):
		return GB
	case size < int64(PB):
		return TB
	case size < int64(EB):
		return PB
	default:
		return EB
	}
}

func BinaryUnitForSize(size int64) SizeUnit {
	switch {
	case size < int64(KiB):
		return Byte
	case size < int64(MiB):
		return KiB
	case size < int64(GiB):
		return MiB
	case size < int64(TiB):
		return GiB
	case size < int64(PiB):
		return TiB
	case size < int64(EiB):
		return PiB
	}
	return EiB
}

func BinarySize(width, precision int, val int64) string {
	unit := BinaryUnitForSize(val)
	return fmt.Sprintf("%[1]*.[2]*[3]f%v", width, precision, unit.Value(val), unit.String())
}

func DecimalSize(width, precision int, val int64) string {
	unit := DecimalUnitForSize(val)
	return fmt.Sprintf("%[1]*.[2]*[3]f%v", width, precision, unit.Value(val), unit.String())
}

func (b SizeUnit) format(f fmt.State, verb rune, binary bool) {
	width, ok := f.Width()
	if !ok {
		width = 0
	}
	prec, ok := f.Precision()
	if !ok {
		prec = 2
	}
	var u SizeUnit
	if binary {
		u = BinaryUnitForSize(int64(b))
	} else {
		u = DecimalUnitForSize(int64(b))
	}
	v := u.Value(int64(b))
	switch verb {
	case 'f':
		fmt.Fprintf(f, "%*.*f %s", width, prec, v, u)
	case 'g', 'G':
		if prec == 0 {
			fmt.Fprintf(f, "%*d %s", width, int(v), u)
		} else {
			fmt.Fprintf(f, "%*.*g %s", width, prec, v, u)
		}
	default:
		fmt.Fprintf(f, "%*.*f %s", width, prec, v, u)
	}
}
