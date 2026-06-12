// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"os"
	"reflect"
)

// ExpandEnv recursively expands environment variables in the
// fields of the provided struct that have a 'yaml' tag. Embedded
// structs are also processed.
// The provided mapping is used to look up variable values.
func Expand(cfg any, mapping func(string) string) {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	expandStruct(v, mapping)
}

func expandStruct(v reflect.Value, mapping func(string) string) {
	t := v.Type()
	for i := range t.NumField() {
		fv := v.Field(i)
		field := t.Field(i)
		if field.Anonymous {
			switch fv.Kind() {
			case reflect.Struct:
				expandStruct(fv, mapping)
			case reflect.Pointer:
				if fv.Type().Elem().Kind() == reflect.Struct && !fv.IsNil() {
					expandStruct(fv.Elem(), mapping)
				}
			}
			continue
		}
		if _, ok := field.Tag.Lookup("yaml"); !ok {
			continue
		}
		expandValue(fv, mapping)
	}
}

func expandValue(v reflect.Value, mapping func(string) string) {
	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			v.SetString(os.Expand(v.String(), mapping))
		}
	case reflect.Struct:
		expandStruct(v, mapping)
	case reflect.Pointer:
		if !v.IsNil() {
			expandValue(v.Elem(), mapping)
		}
	case reflect.Slice:
		for i := range v.Len() {
			expandValue(v.Index(i), mapping)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			elem := v.MapIndex(key)
			// Map values are not addressable; copy via a new value.
			tmp := reflect.New(elem.Type()).Elem()
			tmp.Set(elem)
			expandValue(tmp, mapping)
			v.SetMapIndex(key, tmp)
		}
	}
}
