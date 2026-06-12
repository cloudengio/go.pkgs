// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Variables accumulates scalar key-value pairs parsed from YAML mappings.
// Multiple calls to Load merge into the same map; later values overwrite
// earlier ones for duplicate keys.
type Variables struct {
	vars map[string]string
}

func NewVariables() *Variables {
	return &Variables{
		vars: make(map[string]string),
	}
}

// Mapping returns the value stored for key, or "" if key is not present.
// It is safe to call on a nil or zero-value Variables.
func (v *Variables) Mapping(key string) string {
	if v == nil || len(v.vars) == 0 {
		return ""
	}
	return v.vars[key]
}

// Load parses spec, locates the top-level YAML mapping named mapName, and
// merges its entries into v. All values must be scalar (string, number, or
// boolean); aggregate types (mappings, sequences) are rejected with an error.
// If mapName is not present in spec Load is a no-op.
func (v *Variables) Load(spec []byte, mapName string) error {
	var top map[string]any
	if err := yaml.Unmarshal(spec, &top); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}
	raw, ok := top[mapName]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("%q is not a YAML mapping", mapName)
	}
	if v.vars == nil {
		v.vars = make(map[string]string, len(m))
	}
	for k, val := range m {
		s, err := scalarToString(mapName, k, val)
		if err != nil {
			return err
		}
		v.vars[k] = s
	}
	return nil
}

func scalarToString(mapName, key string, v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case int:
		return strconv.Itoa(val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("%q: value for key %q must be a scalar (string, number, or boolean), got %T", mapName, key, v)
	}
}
