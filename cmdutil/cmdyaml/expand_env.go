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
// The provided envFunc is used to look up environment variable values.
func ExpandEnv(cfg any, envFunc func(string) string) {
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
	expandEnvStruct(v, envFunc)
}

func expandEnvStruct(v reflect.Value, envFunc func(string) string) {
	t := v.Type()
	for i := range t.NumField() {
		fv := v.Field(i)
		field := t.Field(i)
		if field.Anonymous {
			switch fv.Kind() {
			case reflect.Struct:
				expandEnvStruct(fv, envFunc)
			case reflect.Pointer:
				if fv.Type().Elem().Kind() == reflect.Struct && !fv.IsNil() {
					expandEnvStruct(fv.Elem(), envFunc)
				}
			}
			continue
		}
		if _, ok := field.Tag.Lookup("yaml"); !ok {
			continue
		}
		expandEnvValue(fv, envFunc)
	}
}

func expandEnvValue(v reflect.Value, envFunc func(string) string) {
	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			v.SetString(os.Expand(v.String(), envFunc))
		}
	case reflect.Struct:
		expandEnvStruct(v, envFunc)
	case reflect.Pointer:
		if !v.IsNil() {
			expandEnvValue(v.Elem(), envFunc)
		}
	case reflect.Slice:
		for i := range v.Len() {
			expandEnvValue(v.Index(i), envFunc)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			elem := v.MapIndex(key)
			// Map values are not addressable; copy via a new value.
			tmp := reflect.New(elem.Type()).Elem()
			tmp.Set(elem)
			expandEnvValue(tmp, envFunc)
			v.SetMapIndex(key, tmp)
		}
	}
}
