// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Deferred represents a YAML node that has been captured for deferred decoding.
type Deferred yaml.Node

// UnmarshalYAML captures the raw YAML node for deferred decoding.
func (d *Deferred) UnmarshalYAML(value *yaml.Node) error {
	if d == nil {
		return fmt.Errorf("nil Deferred node")
	}
	*d = Deferred(*value)
	return nil
}

// MarshalYAML marshals Deferred as the underlying YAML node.
func (d Deferred) MarshalYAML() (any, error) {
	return (*yaml.Node)(&d), nil
}

// Decode decodes the captured YAML node into the provided value.
func (d *Deferred) Decode(v any) error {
	if d == nil {
		return fmt.Errorf("nil Deferred node")
	}
	return (*yaml.Node)(d).Decode(v)
}

// ValueFor retrieves the value associated with the specified key from a
// mapping node.
func (d *Deferred) ValueFor(key string) (*yaml.Node, bool) {
	if d == nil || d.Kind != yaml.MappingNode {
		return nil, false
	}
	for i := 0; i+1 < len(d.Content); i += 2 {
		if d.Content[i].Value == key {
			return d.Content[i+1], true
		}
	}
	return nil, false
}

// ParseDeferred decodes the provided Deferred YAML node into a value of type T.
func ParseDeferred[T any](d *Deferred) (T, error) {
	var val T
	if d == nil {
		return val, fmt.Errorf("nil Deferred node")
	}
	node := (*yaml.Node)(d)
	if err := node.Decode(&val); err != nil {
		return val, err
	}
	return val, nil
}
