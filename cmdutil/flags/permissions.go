// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"io/fs"

	"cloudeng.io/cmdutil/cmdtypes"
)

// Permissions represents file permission bits that can be used as a
// flag.Value, accepting the same notations as cmdtypes.Permissions: an octal
// number ("0700", "700", "0o700", "0x1c0"), an rwx string ("rwxr-xr-x",
// "-rwx------", "rwx"), or symbolic chmod notation ("u=rwx,go=",
// "u=rwx,go=rx").
type Permissions struct {
	opt   string
	value cmdtypes.Permissions
	set   bool
}

// Set implements flag.Value.
func (p *Permissions) Set(v string) error {
	perm, err := cmdtypes.ParsePermissions(v)
	if err != nil {
		return err
	}
	p.opt = v
	p.value = perm
	p.set = true
	return nil
}

// String implements flag.Value. It returns the permissions as supplied, so
// that a default retains the notation it was written in.
func (p *Permissions) String() string {
	return p.opt
}

// Get implements flag.Getter, returning the fs.FileMode represented by the
// flag.
func (p *Permissions) Get() any {
	return p.value.FileMode()
}

// IsDefault returns true if the value has not been set.
func (p *Permissions) IsDefault() bool {
	return !p.set
}

// FileMode returns the fs.FileMode represented by the flag.
func (p *Permissions) FileMode() fs.FileMode {
	return p.value.FileMode()
}

// Permissions returns the value as a cmdtypes.Permissions, for use where the
// same value is also read from or written to JSON or YAML.
func (p *Permissions) Permissions() cmdtypes.Permissions {
	return p.value
}

// Octal returns the 4 digit octal representation of the permissions
// (e.g. "0700", "4755"), whatever the notation they were supplied in. It is
// the encoding used by cmdtypes.Permissions for text, JSON and YAML, so that a
// value written from a flag reads back as the same value.
func (p *Permissions) Octal() string {
	return p.value.String()
}
