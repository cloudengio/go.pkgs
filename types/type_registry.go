// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package types

import (
	"sync"
)

// Registry records types by name so that new instances of them can be
// constructed given the name of the type. Any number of types may be
// registered with a single Registry. It is safe for concurrent use and its
// zero value is ready for use; it must not be copied after first use.
type Registry sync.Map

func (r *Registry) sm() *sync.Map {
	return (*sync.Map)(r)
}

// RegisterType registers T under the name reported by TypeName[T], replacing
// any type previously registered under that name. New will construct a PT,
// that is a *T, for that name.
func (r *Registry) RegisterType[T any, PT *T]() {
	fn := func() any {
		return PT(new(T))
	}
	r.sm().Store(TypeName[T](), fn)
}

// New returns a newly allocated instance of the type registered under
// typeName, as a pointer to that type, or false if no type is registered
// under that name. Each call returns a distinct instance.
func (r *Registry) New(typeName string) (any, bool) {
	raw, ok := r.sm().Load(typeName)
	if !ok {
		return nil, false
	}
	fn := raw.(func() any)
	return fn(), true
}
