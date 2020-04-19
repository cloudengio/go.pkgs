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
)

// Field represents the description of a field and any similarly tagged
// subfields.
type Field struct {
	// Name is the name of the original field. The name takes
	// into account any name specified via a json or yaml tag.
	Name string
	// Doc is the text extracted from the struct tag for this field.
	Doc string
	// Fields, if this field is a struct, contains descriptions for
	// any documented fields in that struct.
	Fields []Field `json:",omitempty" yaml:",omitempty"`
}

func describeTags(tagName string, typ reflect.Type) []Field {
	var fields []Field
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
		if field.Type.Kind() == reflect.Struct {
			subFields = describeTags(tagName, field.Type)
		}
		if !ok && (len(subFields) == 0) {
			continue
		}
		fields = append(fields, Field{Name: name, Doc: doc, Fields: subFields})
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

func describeFields(indent int, fields []Field) string {
	max := 0
	for _, field := range fields {
		if l := len(field.Name); l > max {
			max = l
		}
	}
	out := &strings.Builder{}
	spaces := strings.Repeat(" ", indent)
	for _, field := range fields {
		out.WriteString(spaces)
		out.WriteString(field.Name)
		out.WriteString(":")
		out.WriteString(strings.Repeat(" ", max-len(field.Name)+1))
		out.WriteString(field.Doc)
		out.WriteString("\n")
		if len(field.Fields) > 0 {
			out.WriteString(describeFields(indent+2, field.Fields))
		}
	}
	return out.String()
}

// String returns a string representation of the description.
func (d *Description) String() string {
	out := &strings.Builder{}
	out.WriteString(d.Detail)
	out.WriteString(describeFields(0, d.Fields))
	return out.String()
}

// TypeName returns the fully qualified name of the supplied type or
// the string representation of an anonymous type.
func TypeName(t interface{}) string {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
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
func Describe(t interface{}, tag, detail string) (*Description, error) {
	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%T is not a struct", t)
	}
	return &Description{
		Detail: detail,
		Fields: describeTags(tag, typ),
	}, nil
}
