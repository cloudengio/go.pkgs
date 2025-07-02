// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"strings"
)

// Repeating represents the values from multiple instances of the same
// command line argument.
type Repeating struct {
	Values   []string
	Validate func(string) error
}

// String inplements flag.Value.
func (r *Repeating) String() string {
	return strings.Join(r.Values, ", ")
}

// Set inplements flag.Value.
func (r *Repeating) Set(v string) error {
	if fn := r.Validate; fn != nil {
		if err := fn(v); err != nil {
			return err
		}
	}
	r.Values = append(r.Values, v)
	return nil
}

// Get inplements flag.Getter.
func (r *Repeating) Get() any {
	return r.Values
}

// Commas represents the values for flags that contain comma separated
// values. The optional validate function is applied to each sub value
// separately.
type Commas struct {
	Values   []string
	Validate func(string) error
}

// Set implements flag.Value.
func (c *Commas) Set(v string) error {
	vals := strings.Split(v, ",")
	if fn := c.Validate; fn != nil {
		for _, val := range vals {
			if err := fn(val); err != nil {
				return err
			}
		}
	}
	c.Values = append(c.Values, vals...)
	return nil
}

// String inplements flag.Value.
func (c *Commas) String() string {
	return strings.Join(c.Values, ", ")
}
