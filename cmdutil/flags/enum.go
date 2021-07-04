// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"fmt"
	"sort"
	"strings"
)

// Map represents a mapping of strings to values that implements flag.Value
// and can be used for command line flag values. It must be appropriately
// initialized with name, value pairs and a default value using its
// Register and Default methods.
type Map struct {
	value  interface{}
	values map[string]interface{}
}

type ErrMap struct {
	msg string
}

// Error implements error.
func (me *ErrMap) Error() string {
	return me.msg
}

// Is implements errors.Is.
func (me ErrMap) Is(target error) bool {
	_, ok := target.(*ErrMap)
	return ok
}

// Set implements flag.Value.
func (ef *Map) Set(v string) error {
	if ef.values == nil {
		return &ErrMap{msg: "no values have been registered"}
	}
	tmp, ok := ef.values[v]
	if !ok {
		vals := make([]string, 0, len(ef.values))
		for k := range ef.values {
			vals = append(vals, k)
		}
		sort.Strings(vals)
		return &ErrMap{msg: fmt.Sprintf("%v not one of %v", v, strings.Join(vals, ", "))}
	}
	ef.value = tmp
	return nil
}

// String implements flag.Value.
func (ef *Map) String() string {
	for k, v := range ef.values {
		if v == ef.value {
			return k
		}
	}
	if ef.value == nil {
		return ""
	}
	return fmt.Sprintf("%v", ef.value)
}

// Value implements flag.Getter.
func (ef *Map) Get() interface{} {
	return ef.value
}

func (ef Map) Register(name string, val interface{}) Map {
	if ef.values == nil {
		ef.values = map[string]interface{}{}
	}
	ef.values[name] = val
	return ef
}

func (ef Map) Default(val interface{}) Map {
	if ef.values == nil {
		ef.values = map[string]interface{}{}
	}
	ef.value = val
	return ef
}
