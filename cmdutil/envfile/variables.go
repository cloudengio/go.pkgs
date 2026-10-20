// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile

import (
	"io"
	"maps"
	"os"
)

// Variables holds the environment variable definitions read from one or more
// envfiles. Its Mapping method has the signature expected for the mapping
// argument of os.Expand, and of the many APIs that accept such a function,
// so the definitions in an envfile can be used to expand $VAR and ${VAR}
// references anywhere one is called for:
//
//	os.Expand(s, vars.Mapping)
//
// Note that Parse, and hence Read, does not itself expand references that
// appear within an envfile's own values; use Expand, or Mapping, to resolve
// those.
//
// The read-only methods are safe to call on a nil or zero-value Variables,
// which behaves as one with no definitions. Merge and MergeFile require a
// non-nil receiver.
type Variables struct {
	vars map[string]string
}

// NewVariables returns a Variables containing a copy of vars.
func NewVariables(vars map[string]string) *Variables {
	return &Variables{vars: maps.Clone(vars)}
}

// Read reads environment variable definitions from r, as parsed by Parse,
// and returns them as a Variables.
func Read(r io.Reader) (*Variables, error) {
	vars, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return &Variables{vars: vars}, nil
}

// ReadFile is a convenience wrapper around Read that opens and reads a file.
func ReadFile(filename string) (*Variables, error) {
	vars, err := ParseFile(filename)
	if err != nil {
		return nil, err
	}
	return &Variables{vars: vars}, nil
}

// Mapping returns the value defined for name, or "" if name is not defined.
// Its signature matches the mapping argument of os.Expand. Use Lookup to
// distinguish a variable defined as empty from one that is not defined.
func (v *Variables) Mapping(name string) string {
	if v == nil {
		return ""
	}
	return v.vars[name]
}

// Lookup returns the value defined for name and whether it is defined.
func (v *Variables) Lookup(name string) (string, bool) {
	if v == nil {
		return "", false
	}
	val, ok := v.vars[name]
	return val, ok
}

// Expand is shorthand for os.Expand(s, v.Mapping) and hence replaces
// references to undefined variables with "" and expands s in a single pass,
// leaving any references in the values it substitutes unexpanded.
func (v *Variables) Expand(s string) string {
	return os.Expand(s, v.Mapping)
}

// Merge reads further definitions from r and merges them into v, with the
// values read from r overwriting any already defined for the same name. If r
// cannot be parsed v is left unchanged.
func (v *Variables) Merge(r io.Reader) error {
	vars, err := Parse(r)
	if err != nil {
		return err
	}
	v.merge(vars)
	return nil
}

// MergeFile is a convenience wrapper around Merge that opens and reads a file.
func (v *Variables) MergeFile(filename string) error {
	vars, err := ParseFile(filename)
	if err != nil {
		return err
	}
	v.merge(vars)
	return nil
}

func (v *Variables) merge(vars map[string]string) {
	if v.vars == nil {
		v.vars = make(map[string]string, len(vars))
	}
	maps.Copy(v.vars, vars)
}

// All returns a copy of the definitions held by v.
func (v *Variables) All() map[string]string {
	if v == nil {
		return nil
	}
	return maps.Clone(v.vars)
}

// Len returns the number of variables defined.
func (v *Variables) Len() int {
	if v == nil {
		return 0
	}
	return len(v.vars)
}
