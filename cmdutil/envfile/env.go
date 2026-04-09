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
// the `use_env` and `use_env_file` struct tags. It caches parsed envfiles across
// multiple calls to Expand so each file is read at most once.
type StructEnv struct {
	fileCache map[string]map[string]string
}

// Expand processes the exported string fields of the struct pointed to by s
// and expands environment variable references in those fields.
//
// Two struct tags are recognised:
//
//   - `use_env`: the field's current value may contain $VAR or ${VAR} references
//     that are expanded using the process environment (os.LookupEnv).
//
//   - `use_env_file`: the field's current value encodes both the source file and
//     the variable reference in the form:
//
//     filename:$VAR
//     filename:${VAR}
//
//     The filename is parsed greedily: it is everything before the last ':'
//     that is immediately followed by '$'. This allows filenames that contain
//     ':' (e.g. absolute Windows paths). The file is parsed with ParseFile
//     and the variable reference is expanded using the resulting map.
//     If the field value does not match the pattern it is left unchanged.
//
// Both tags use the same ${VAR} / $VAR syntax as os.Expand. Non-string
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

		if _, ok := field.Tag.Lookup("use_env"); ok {
			fv.SetString(os.Expand(fv.String(), os.Getenv))
			continue
		}

		if _, ok := field.Tag.Lookup("use_env_file"); ok {
			filename, ref, found := splitFilenameRef(fv.String())
			if !found {
				continue
			}
			vars, err := se.cachedParseFile(filename)
			if err != nil {
				return fmt.Errorf("Expand: field %s: envfile %q: %w", field.Name, filename, err)
			}
			fv.SetString(os.Expand(ref, func(key string) string {
				return vars[key]
			}))
		}
	}
	return nil
}

// splitFilenameRef splits a field value of the form "filename:$VAR" or
// "filename:${VAR}" into the filename and the variable reference.
// The filename is parsed greedily: it is everything before the last ':'
// that is immediately followed by '$', so filenames containing ':' work
// correctly. Returns ok=false if no such pattern is found.
func splitFilenameRef(s string) (filename, ref string, ok bool) {
	last := -1
	for i := range len(s) - 1 {
		if s[i] == ':' && s[i+1] == '$' {
			last = i
		}
	}
	if last < 0 {
		return "", "", false
	}
	return s[:last], s[last+1:], true
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
