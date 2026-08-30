// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package types provides support for working with go types.
package types

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// typeNameCache maps reflect.Type → string. reflect.Type is a pointer-backed
// interface, so using it as a sync.Map key is allocation-free on Load.
var typeNameCache sync.Map

// TypeName returns the fully qualified type name of type T
// (e.g. "example.com/pkg.MyType"). Pointer indirection is removed
// so that TypeName[*MyType]() == TypeName[MyType](). A composite type
// that has no name of its own, such as []MyType, is described in terms
// of the fully qualified names of the types it is built from
// (e.g. "[]example.com/pkg.MyType"). Results are computed once per
// distinct concrete type and cached.
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
// A composite type that has no name of its own is described in terms of
// the fully qualified names of the types it is built from, as per
// TypeName. The empty string is returned only for an untyped nil, which
// carries no type. Results are computed once per distinct concrete type
// and cached.
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
	var sb strings.Builder
	writeTypeName(&sb, typ)
	name := sb.String()
	typeNameCache.Store(typ, name)
	return name
}

// writeTypeName writes the fully qualified name of typ to sb. Defined and
// predeclared types are named directly; a composite type that has no name of
// its own is described structurally, in terms of the fully qualified names of
// the types it is built from, so that distinct types always have distinct
// names.
func writeTypeName(sb *strings.Builder, typ reflect.Type) {
	if name := typ.Name(); name != "" {
		if pkg := typ.PkgPath(); pkg != "" {
			sb.WriteString(pkg)
			sb.WriteByte('.')
		}
		sb.WriteString(name)
		return
	}
	switch typ.Kind() {
	case reflect.Pointer:
		sb.WriteByte('*')
		writeTypeName(sb, typ.Elem())
	case reflect.Slice:
		sb.WriteString("[]")
		writeTypeName(sb, typ.Elem())
	case reflect.Array:
		sb.WriteByte('[')
		sb.WriteString(strconv.Itoa(typ.Len()))
		sb.WriteByte(']')
		writeTypeName(sb, typ.Elem())
	case reflect.Map:
		sb.WriteString("map[")
		writeTypeName(sb, typ.Key())
		sb.WriteByte(']')
		writeTypeName(sb, typ.Elem())
	case reflect.Chan:
		switch typ.ChanDir() {
		case reflect.RecvDir:
			sb.WriteString("<-chan ")
		case reflect.SendDir:
			sb.WriteString("chan<- ")
		default:
			sb.WriteString("chan ")
		}
		writeTypeName(sb, typ.Elem())
	case reflect.Func:
		sb.WriteString("func")
		writeSignature(sb, typ)
	case reflect.Struct:
		writeStructType(sb, typ)
	case reflect.Interface:
		writeInterfaceType(sb, typ)
	default:
		// Every unnamed kind is handled above; fall back to reflect's own
		// rendering rather than to nothing at all.
		sb.WriteString(typ.String())
	}
}

// writeSignature writes the parameters and results of the function type typ,
// that is, its type with the leading func omitted.
func writeSignature(sb *strings.Builder, typ reflect.Type) {
	sb.WriteByte('(')
	for i := range typ.NumIn() {
		if i > 0 {
			sb.WriteString(", ")
		}
		if typ.IsVariadic() && i == typ.NumIn()-1 {
			sb.WriteString("...")
			writeTypeName(sb, typ.In(i).Elem())
			continue
		}
		writeTypeName(sb, typ.In(i))
	}
	sb.WriteByte(')')
	switch typ.NumOut() {
	case 0:
	case 1:
		sb.WriteByte(' ')
		writeTypeName(sb, typ.Out(0))
	default:
		sb.WriteString(" (")
		for i := range typ.NumOut() {
			if i > 0 {
				sb.WriteString(", ")
			}
			writeTypeName(sb, typ.Out(i))
		}
		sb.WriteByte(')')
	}
}

func writeStructType(sb *strings.Builder, typ reflect.Type) {
	if typ.NumField() == 0 {
		sb.WriteString("struct {}")
		return
	}
	sb.WriteString("struct {")
	for i := range typ.NumField() {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteByte(' ')
		f := typ.Field(i)
		if !f.Anonymous {
			sb.WriteString(f.Name)
			sb.WriteByte(' ')
		}
		writeTypeName(sb, f.Type)
	}
	sb.WriteString(" }")
}

func writeInterfaceType(sb *strings.Builder, typ reflect.Type) {
	if typ.NumMethod() == 0 {
		sb.WriteString("interface {}")
		return
	}
	sb.WriteString("interface {")
	for i := range typ.NumMethod() {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteByte(' ')
		m := typ.Method(i)
		if m.PkgPath != "" {
			sb.WriteString(m.PkgPath)
			sb.WriteByte('.')
		}
		sb.WriteString(m.Name)
		writeSignature(sb, m.Type)
	}
	sb.WriteString(" }")
}
