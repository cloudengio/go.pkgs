// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile

import (
	"fmt"
	"os"
	"reflect"
)

// StructEnv expands environment variable references in struct fields using
// the `env` and `envfile` struct tags. It caches parsed envfiles across
// multiple calls to Expand so each file is read at most once.
type StructEnv struct {
	fileCache map[string]map[string]string
}

// Expand processes the exported string fields of the struct pointed to by s
// and expands environment variable references in those fields.
//
// Two struct tags are recognised:
//
//   - `env`: the field's current value may contain $VAR or ${VAR} references
//     that are expanded using the process environment (os.LookupEnv).
//
//   - `envfile:"filename"`: the named file is parsed with ParseFile and
//     $VAR or ${VAR} references in the field's current value are expanded
//     using the variables defined in that file.
//
// Both tags use the same ${VAR} / $VAR syntax as os.Expand. A field value
// that contains no $ is treated as a literal and left unchanged. Non-string
// fields are silently skipped regardless of tags.
//
// The struct must be passed as a non-nil pointer.
func (se *StructEnv) Expand(s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("Expand: requires a non-nil pointer to a struct, got %T", s)
	}
	v = v.Elem()
	t := v.Type()

	if se.fileCache == nil {
		se.fileCache = map[string]map[string]string{}
	}

	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() || fv.Kind() != reflect.String {
			continue
		}

		if _, ok := field.Tag.Lookup("env"); ok {
			fv.SetString(os.Expand(fv.String(), os.Getenv))
			continue
		}

		if filename, ok := field.Tag.Lookup("envfile"); ok {
			vars, err := se.cachedParseFile(filename)
			if err != nil {
				return fmt.Errorf("Expand: field %s: envfile %q: %w", field.Name, filename, err)
			}
			fv.SetString(os.Expand(fv.String(), func(key string) string {
				return vars[key]
			}))
		}
	}
	return nil
}

func (se *StructEnv) cachedParseFile(filename string) (map[string]string, error) {
	if vars, ok := se.fileCache[filename]; ok {
		return vars, nil
	}
	vars, err := ParseFile(filename)
	if err != nil {
		return nil, err
	}
	se.fileCache[filename] = vars
	return vars, nil
}
