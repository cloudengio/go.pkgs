// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package structdoc provides a means of exposing struct tags for use when
// generating documentation for those structs.
package structdoc

import (
	"fmt"
	"reflect"
	"strings"

	"cloudeng.io/text/linewrap"
)

// Field represents the description of a field and any similarly tagged
// subfields.
type Field struct {
	// Name is the name of the original field. The name takes
	// into account any name specified via a json or yaml tag.
	Name string
	// Doc is the text extracted from the struct tag for this field.
	Doc string
	// Slice is true if this field is a slice.
	Slice bool
	// Fields, if this field is a struct, contains descriptions for
	// any documented fields in that struct.
	Fields []Field `json:",omitempty" yaml:",omitempty"`
}

func describeTags(seen map[reflect.Type]struct{}, tagName string, typ reflect.Type) []Field {
	if _, ok := seen[typ]; ok {
		return nil
	}
	seen[typ] = struct{}{}
	var fields []Field
	if typ.Kind() != reflect.Struct {
		return nil
	}
	for nf := 0; nf < typ.NumField(); nf++ {
		field := typ.Field(nf)
		doc, ok := field.Tag.Lookup(tagName)
		name := field.Name
		// Heurestic to use the same name as any other intended encoding
		// for this field.
		for _, encoding := range []string{"yaml", "json"} {
			if etag, ok := field.Tag.Lookup(encoding); ok {
				if parts := strings.Split(etag, ","); len(parts) > 0 {
					name = parts[0]
				}
			}
		}
		var subFields []Field
		var slice bool
		if field.Type.Kind() == reflect.Struct {
			subFields = describeTags(seen, tagName, field.Type)
			if field.Anonymous {
				fields = append(fields, subFields...)
				subFields = nil
			}
		}
		if field.Type.Kind() == reflect.Slice {
			slice = true
			subFields = describeTags(seen, tagName, field.Type.Elem())
		}
		if !ok && (len(subFields) == 0) {
			continue
		}
		fields = append(fields, Field{Name: name, Doc: doc, Slice: slice, Fields: subFields})
	}
	return fields
}

// Description represents a structured description of a struct type based
// on struct tags. The Detail field may be supplied when constructing
// the description.
type Description struct {
	Detail string
	Fields []Field
}

// FormatFields formats the supplied fields as follows:
//
//	<prefix><name>:<padding><text>
//
// where padding is calculated so as to line up the text. Prefix sets the
// number of spaces to be prefixed and indent increases the prefix for each
// sub field.
func FormatFields(prefix, indent int, fields []Field) string {
	maxf := 0
	for _, field := range fields {
		if l := len(field.Name); l > maxf {
			maxf = l
		}
	}
	out := &strings.Builder{}
	spaces := strings.Repeat(" ", prefix)
	for _, field := range fields {
		if len(field.Name) > 0 {
			doc := field.Doc
			if field.Slice {
				doc = "[]" + doc
			}
			doc = linewrap.Paragraph(maxf-len(field.Name)+1, maxf+prefix+2, 80, doc)
			out.WriteString(spaces)
			out.WriteString(field.Name)
			out.WriteString(":")
			out.WriteString(doc)
			out.WriteString("\n")
		}
		if len(field.Fields) > 0 {
			out.WriteString(FormatFields(prefix+indent, indent, field.Fields))
		}
	}
	return out.String()
}

// String returns a string representation of the description.
func (d *Description) String() string {
	out := &strings.Builder{}
	out.WriteString(d.Detail)
	out.WriteString(FormatFields(0, 2, d.Fields))
	return out.String()
}

// TypeName returns the fully qualified name of the supplied type or
// the string representation of an anonymous type.
func TypeName(t any) string {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if name := typ.Name(); len(name) > 0 {
		return typ.PkgPath() + "." + name
	}
	return typ.String()
}

// Describe generates a Description for the supplied type based on its
// struct tags. Detail can be used to provide a top level of detail,
// such as the type name and a summary.
func Describe(t any, tag, detail string) (*Description, error) {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%T is not a struct", t)
	}
	seen := map[reflect.Type]struct{}{}
	return &Description{
		Detail: detail,
		Fields: describeTags(seen, tag, typ),
	}, nil
}
