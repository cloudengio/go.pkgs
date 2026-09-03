// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import "cloudeng.io/cmdutil/cmdtypes"

// Permissions is an alias for cmdtypes.Permissions, which is decoded from
// YAML via its UnmarshalText method. It is retained here for the convenience
// of code that already refers to it by this name.
type Permissions = cmdtypes.Permissions

// ParsePermissions parses a permission string in octal, rwx, or symbolic
// format, as per cmdtypes.ParsePermissions.
func ParsePermissions(s string) (Permissions, error) {
	return cmdtypes.ParsePermissions(s)
}
