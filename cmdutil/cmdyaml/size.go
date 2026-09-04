// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import "cloudeng.io/cmdutil/cmdtypes"

// ByteSize is an alias for cmdtypes.ByteSize, which is decoded from YAML via
// its UnmarshalText method. It is retained here because other packages refer
// to it by this name.
type ByteSize = cmdtypes.ByteSize

// The byte size units, as per cmdtypes.
const (
	Byte = cmdtypes.Byte

	KB = cmdtypes.KB
	MB = cmdtypes.MB
	GB = cmdtypes.GB
	TB = cmdtypes.TB

	KiB = cmdtypes.KiB
	MiB = cmdtypes.MiB
	GiB = cmdtypes.GiB
	TiB = cmdtypes.TiB
)

// ParseByteSize parses s into a ByteSize, as per cmdtypes.ParseByteSize.
func ParseByteSize(s string) (ByteSize, error) {
	return cmdtypes.ParseByteSize(s)
}
