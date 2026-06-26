// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Regexp wraps a *regexp.Regexp so that it can be marshaled to and
// unmarshaled from YAML as the regular expression's source pattern string.
// The zero value has a nil *regexp.Regexp.
type Regexp struct {
	*regexp.Regexp
}

// String returns the source text of the regular expression, or "" if r
// wraps a nil *regexp.Regexp.
func (r Regexp) String() string {
	if r.Regexp == nil {
		return ""
	}
	return r.Regexp.String()
}

// MarshalYAML implements yaml.Marshaler, encoding r as its source pattern
// string.
func (r Regexp) MarshalYAML() (any, error) {
	return r.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler, compiling the YAML scalar
// string value as a regular expression.
func (r *Regexp) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return fmt.Errorf("invalid regular expression %q: %w", s, err)
	}
	r.Regexp = re
	return nil
}

// RegexpList is a list of Regexp values that can be marshaled to and
// unmarshaled from a YAML sequence of regular expression strings.
type RegexpList []Regexp

// MarshalYAML implements yaml.Marshaler, encoding rl as a sequence of
// source pattern strings.
func (rl RegexpList) MarshalYAML() (any, error) {
	out := make([]string, len(rl))
	for i, r := range rl {
		out[i] = r.String()
	}
	return out, nil
}

// UnmarshalYAML implements yaml.Unmarshaler, compiling each element of the
// YAML sequence as a regular expression.
func (rl *RegexpList) UnmarshalYAML(value *yaml.Node) error {
	var exprs []string
	if err := value.Decode(&exprs); err != nil {
		return err
	}
	list := make(RegexpList, len(exprs))
	for i, expr := range exprs {
		re, err := regexp.Compile(expr)
		if err != nil {
			return fmt.Errorf("invalid regular expression %q: %w", expr, err)
		}
		list[i] = Regexp{re}
	}
	*rl = list
	return nil
}

// Regexps returns a slice of the *regexp.Regexp values in rl.
func (rl RegexpList) Regexps() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(rl))
	for i, r := range rl {
		out[i] = r.Regexp
	}
	return out
}
