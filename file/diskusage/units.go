// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package diskusage

type Base2Bytes int64

const (
	KiB Base2Bytes = 1024
	MiB Base2Bytes = KiB * 1024
	GiB Base2Bytes = MiB * 1024
	TiB Base2Bytes = GiB * 1024
	PiB Base2Bytes = TiB * 1024
	EiB Base2Bytes = PiB * 1024
)

type DecimalBytes int64

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
