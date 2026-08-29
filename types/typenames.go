// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package types provides support for working with gp types.
package types

import (
	"reflect"
	"sync"
)

// typeNameCache maps reflect.Type → string. reflect.Type is a pointer-backed
// interface, so using it as a sync.Map key is allocation-free on Load.
var typeNameCache sync.Map

// TypeName returns the fully qualified type name of type T
// (e.g. "example.com/pkg.MyType"). Pointer indirection is removed
// so that TypeName[*MyType]() == TypeName[MyType](). Results are
// computed once per distinct concrete type and cached.
func TypeName[T any]() string {
	typ := reflect.TypeFor[T]()
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return cachedTypeName(typ)
}

// TypeNameForValue returns the fully qualified type name of v
// (e.g. "example.com/pkg.MyType"). Pointer indirection is removed
// so that TypeNameForValue(&MyType{}) == TypeNameForValue(MyType{}).
// Results are computed once per distinct concrete type and cached.
func TypeNameForValue(v any) string {
	typ := reflect.TypeOf(v)
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return cachedTypeName(typ)
}

func cachedTypeName(typ reflect.Type) string {
	if typ == nil {
		return ""
	}
	if v, ok := typeNameCache.Load(typ); ok {
		return v.(string)
	}
	var name string
	if typ.PkgPath() == "" {
		name = typ.Name()
	} else {
		name = typ.PkgPath() + "." + typ.Name()
	}
	typeNameCache.Store(typ, name)
	return name
}
