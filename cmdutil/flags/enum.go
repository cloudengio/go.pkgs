// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"fmt"
	"sort"
	"strings"
)

// EnumValues must be implemented by any type that is to be used with Enum.
type EnumValues[T any] interface {
	EnumValues() map[string]T
}

// EnumType combines comparable underlying types with EnumSpec.
type EnumType[T any] interface {
	comparable
	EnumValues[T]
}

// Enum is a generic flag.Value wrapper that supports enumerated types.
type Enum[T EnumType[T]] struct {
	Value T
}

// String implements flag.Value
func (e Enum[T]) String() string {
	opts := e.Value.EnumValues()
	for str, val := range opts {
		if val == e.Value {
			return str
		}
	}
	return fmt.Sprintf("%v", e.Value)
}

// AllowedValues returns a sorted, comma-separated list of all valid string
// representations for this Enum.
func (e Enum[T]) AllowedValues() string {
	opts := e.Value.EnumValues()
	vals := make([]string, 0, len(opts))
	for k := range opts {
		vals = append(vals, k)
	}
	sort.Strings(vals)
	return strings.Join(vals, ", ")
}

// MarshalText implements encoding.TextMarshaler.
func (e Enum[T]) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (e *Enum[T]) UnmarshalText(text []byte) error {
	return e.Set(string(text))
}

// Set implements flag.Value and returns available options on invalid input.
func (e *Enum[T]) Set(val string) error {
	opts := e.Value.EnumValues()
	parsed, ok := opts[val]
	if !ok {
		return fmt.Errorf("invalid value %q (must be one of: %s)", val, e.AllowedValues())
	}
	e.Value = parsed
	return nil
}
