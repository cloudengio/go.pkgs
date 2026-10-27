// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"cloudeng.io/errors"
)

// DefaultsMapping returns a mapping, with the signature expected by
// os.Expand, that takes its values from defaults, which it copies. It is
// intended to come after the other mappings in a chain, see ChainMapping, so
// that it supplies a value only for a variable that none of them defines:
//
//	mapping := envfile.ChainMapping(
//		envfile.ExpandingMapping(vars),
//		envfile.DefaultsMapping(map[string]string{"REGION": "us-west-2"}),
//		unresolved.Mapping)
//
// As with any mapping, a default that is empty is indistinguishable from
// none.
func DefaultsMapping(defaults map[string]string) func(string) string {
	defaults = maps.Clone(defaults)
	return func(name string) string {
		return defaults[name]
	}
}

// UnresolvedError is the error recorded by Unresolved for a variable that
// could not be expanded.
type UnresolvedError struct {
	Name string
}

// Error implements error.
func (e *UnresolvedError) Error() string {
	return fmt.Sprintf("variable %s is not defined", e.Name)
}

// Unresolved records the variables that could not be expanded. Its Mapping
// method is intended to be the last in a chain of mappings, see
// ChainMapping, so that it is called only for a variable that none of the
// others defines, which it records, and resolves to nothing. Once the
// expansion is done, Err reports whether any were recorded, and Names
// returns them. The zero value is ready for use, and it is safe for
// concurrent use.
type Unresolved struct {
	mu    sync.Mutex
	names []string
	errs  errors.M
}

// Mapping records name, once, however often it is called for it, and
// returns "".
func (u *Unresolved) Mapping(name string) string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if !slices.Contains(u.names, name) {
		u.names = append(u.names, name)
		u.errs.Append(&UnresolvedError{Name: name})
	}
	return ""
}

// Names returns the names of the variables recorded, in the order in which
// they were first recorded.
func (u *Unresolved) Names() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.names)
}

// Err returns nil if no variable was recorded, or an error that contains an
// *UnresolvedError for each that was, see errors.M.
func (u *Unresolved) Err() error {
	return u.errs.Err()
}
