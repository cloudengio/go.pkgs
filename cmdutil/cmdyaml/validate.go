// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"fmt"
	"reflect"
	"strings"
)

type validator interface{ Validate() error }

var validatorType = reflect.TypeFor[validator]()

// ValidateFields calls the Validate method of each of the values of the fields
// of cfg, which must be a pointer to a struct, that has one, and returns the
// first error, prefixed with the YAML path to the value, eg.
// tls_config.hosts[2]: reason. It walks fields that are structs, pointers,
// interfaces, slices, arrays and maps, but not into a value that has a Validate
// method, which is responsible for its own fields. Fields that are inlined are
// walked as if they were fields of cfg. It does not call cfg's own Validate
// method, and note that a struct that embeds a type with a Validate method
// has one too, as a promoted method. A Validate method with a pointer receiver
// is called for values that are addressable, which map values are not.
func ValidateFields(cfg any) error {
	v := reflect.ValueOf(cfg)
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("%T is not a struct or a pointer to one", cfg)
	}
	return validateChildren(v, "")
}

func validateChildren(v reflect.Value, path string) error {
	for i := range v.NumField() {
		field := v.Type().Field(i)
		if !field.IsExported() {
			continue
		}
		name, inline := yamlName(field)
		fieldPath := path
		if !inline {
			fieldPath = join(path, name)
		}
		if err := validateValue(v.Field(i), fieldPath); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(v reflect.Value, path string) error {
	if !v.IsValid() || (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && v.IsNil() {
		return nil
	}
	if ok, err := tryValidate(v, path); ok {
		return err
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return validateValue(v.Elem(), path)
	case reflect.Struct:
		return validateChildren(v, path)
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			if err := validateValue(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if err := validateValue(iter.Value(), fmt.Sprintf("%s[%v]", path, iter.Key())); err != nil {
				return err
			}
		}
	}
	return nil
}

// tryValidate calls the Validate method of v, if it has one, and reports
// whether it did.
func tryValidate(v reflect.Value, path string) (bool, error) {
	var val validator
	switch {
	case v.Type().Implements(validatorType):
		if v.Kind() == reflect.Pointer && v.IsNil() {
			return false, nil
		}
		val = v.Interface().(validator)
	case v.CanAddr() && v.Addr().Type().Implements(validatorType):
		val = v.Addr().Interface().(validator)
	default:
		return false, nil
	}
	if err := val.Validate(); err != nil {
		if path == "" {
			return true, err
		}
		return true, fmt.Errorf("%s: %w", path, err)
	}
	return true, nil
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// yamlName returns the name of the YAML key for the field, and whether the
// field is inlined.
func yamlName(f reflect.StructField) (name string, inline bool) {
	tag := f.Tag.Get("yaml")
	parts := strings.Split(tag, ",")
	for _, opt := range parts[1:] {
		if opt == "inline" {
			return "", true
		}
	}
	if parts[0] != "" && parts[0] != "-" {
		return parts[0], false
	}
	return strings.ToLower(f.Name), false
}
