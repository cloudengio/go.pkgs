// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

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
	if v.vars == nil {
		v.vars = make(map[string]string)
	}
	dec := yaml.NewDecoder(bytes.NewReader(spec))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := v.mergeFromNode(&doc, mapName); err != nil {
			return err
		}
	}
}

func (v *Variables) mergeFromNode(node *yaml.Node, mapName string) error {
	if node == nil || node.Kind != yaml.DocumentNode {
		return nil
	}
	for _, child := range node.Content {
		if child.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(child.Content); i += 2 {
			key := child.Content[i]
			val := child.Content[i+1]
			if key.Kind == yaml.ScalarNode && key.Value == mapName {
				if val.Kind != yaml.MappingNode {
					return fmt.Errorf("%q is not a YAML mapping", mapName)
				}
				if err := v.parseVariablesBlock(val, mapName); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// parseVariablesBlock collects the raw key/value pairs of node and resolves
// $VAR and ${VAR} references within their values via repeated passes, so
// that variables may reference one another regardless of the order in which
// they are declared within the block (e.g. ba: ${a} declared before a:
// hello). Each pass re-expands every raw value against the variables
// resolved so far (this block's own entries plus any already accumulated by
// v); it stops once a full pass produces no further changes.
func (v *Variables) parseVariablesBlock(node *yaml.Node, mapName string) error {
	var keys []string
	raw := make(map[string]string)
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode {
			continue
		}
		if valNode.Kind != yaml.ScalarNode {
			return fmt.Errorf("%q: value for key %q must be a scalar", mapName, keyNode.Value)
		}
		if _, ok := raw[keyNode.Value]; !ok {
			keys = append(keys, keyNode.Value)
		}
		raw[keyNode.Value] = valNode.Value
	}

	resolved := make(map[string]string, len(keys))
	lookup := func(name string) string {
		if rs, ok := resolved[name]; ok {
			return rs
		}
		return v.Mapping(name)
	}
	for pass := 0; pass <= len(keys); pass++ {
		changed := false
		for _, key := range keys {
			next := expandVars(raw[key], lookup)
			if next != resolved[key] {
				resolved[key] = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	for _, key := range keys {
		v.vars[key] = resolved[key]
	}
	return nil
}

// expandVars resolves $VAR and ${VAR} references in s using mapping,
// leaving unresolved references untouched.
func expandVars(s string, mapping func(string) string) string {
	return os.Expand(s, func(name string) string {
		rs := mapping(name)
		if len(rs) == 0 {
			return "${" + name + "}"
		}
		return rs
	})
}
