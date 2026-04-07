// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"gopkg.in/yaml.v3"
)

type Deferred struct {
	yaml.Node `yaml:",inline"`
}

// UnmarshalYAML captures the raw YAML node for deferred decoding.
func (d *Deferred) UnmarshalYAML(value *yaml.Node) error {
	d.Node = *value
	return nil
}

func (c Deferred) ValueFor(key string) (yaml.Node, bool) {
	if c.Kind != yaml.MappingNode {
		return yaml.Node{}, false
	}
	for i := 0; i+1 < len(c.Content); i += 2 {
		if c.Content[i].Value == key {
			return *c.Content[i+1], true
		}
	}
	return yaml.Node{}, false
}

/*
func (c *ConfigDeferred) UnmarshalYAML(value *yaml.Node) error {
	/*
		if value.Kind != yaml.MappingNode {
			return fmt.Errorf("expected a mapping node, got %v", value.Kind)
		}
		detail := *value
		detail.Content = nil
		for i := 0; i+1 < len(value.Content); i += 2 {
			if value.Content[i].Value == "type" {
				c.Type = value.Content[i+1].Value
				continue
			}
			detail.Content = append(detail.Content, value.Content[i], value.Content[i+1])
		}
		if c.Type == "" {
			return fmt.Errorf("missing required field 'type'")

		c.Content = detail*
	c.Content = *value
	c.Content.Content = slices.Clone(value.Content)
	return nil
}
*/

func ParseDeferred[T any](d Deferred) (T, error) {
	var val T
	if err := d.Decode(&val); err != nil {
		return val, err
	}
	return val, nil
}
