// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package cmdtypes provides types that are shared by the configuration and
// command line packages, and that can be encoded and decoded as JSON, YAML
// and text without those packages depending on each other.
package cmdtypes

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

// Permissions represents file permission bits (e.g. 0700, 0755). It can be
// decoded from an octal number ("0700", "700", "0o700", "0x1c0"), an rwx
// string ("rwxr-xr-x", "-rwx------", "rwx"), or symbolic chmod notation
// ("u=rwx,go=", "u=rwx,go=rx"), and is always encoded as the 4 digit octal
// string, which every one of those formats can be read back from.
//
// The setuid, setgid and sticky bits are represented as fs.ModeSetuid,
// fs.ModeSetgid and fs.ModeSticky, whichever notation they were written in,
// so that 4755 and rwsr-xr-x yield the same value. They are written out in
// the traditional UNIX form, ie. in the leading octal digit.
//
// Encoding and decoding is implemented by MarshalText and UnmarshalText,
// which both encoding/json and gopkg.in/yaml.v3 use, so a single type serves
// both without this package depending on either. YAML additionally routes
// unquoted numbers such as 0700 through UnmarshalText, so they are read as
// octal rather than as decimal. JSON has no octal literal, so a JSON number
// is handled by UnmarshalJSON below, which reads it as octal for consistency.
type Permissions fs.FileMode

// FileMode returns the fs.FileMode represented by p.
func (p Permissions) FileMode() fs.FileMode {
	return fs.FileMode(p)
}

// String returns the 4 digit octal representation of the permissions
// (e.g. "0700", "4755"), in the traditional UNIX form in which the leading
// digit carries the setuid, setgid and sticky bits.
func (p Permissions) String() string {
	return fmt.Sprintf("%04o", unixBits(fs.FileMode(p)))
}

// MarshalText implements encoding.TextMarshaler, and with it the encoding
// used for both JSON and YAML.
func (p Permissions) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%04o", unixBits(fs.FileMode(p))), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, and with it the decoding
// used for both JSON strings and YAML scalars.
func (p *Permissions) UnmarshalText(text []byte) error {
	perm, err := ParsePermissions(string(text))
	if err != nil {
		return err
	}
	*p = perm
	return nil
}

// UnmarshalJSON implements json.Unmarshaler. It exists only so that a JSON
// number is read as octal, as it is in YAML and on a command line: JSON has no
// octal literal, so 700 would otherwise be read as decimal and grant the wrong
// bits. Strings are handled by UnmarshalText.
func (p *Permissions) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "null" {
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		return p.UnmarshalText([]byte(str))
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("expected a string or number for permissions, got %s: %w", s, err)
	}
	return p.UnmarshalText([]byte(num.String()))
}
