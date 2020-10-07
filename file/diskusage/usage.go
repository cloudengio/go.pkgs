package diskusage

type OnDiskSize func(int64) int64

type RAID0 struct {
	StripeSize int64
	NumStripes int
}

func (r0 RAID0) OnDiskSize(size int64) int64 {
	raw := ((size + r0.StripeSize) / r0.StripeSize) * r0.StripeSize
	striped := int64(r0.NumStripes) * r0.StripeSize
	if striped > raw {
		return striped
	}
	return raw
}

type Simple struct {
	BlockSize int64
}

func (s Simple) OnDiskSize(size int64) int64 {
	return ((size + s.BlockSize) / s.BlockSize) * s.BlockSize
}

type Identity struct{}

func (i Identity) OnDiskSize(size int64) int64 {
	return size
}
