// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"time"

	"gopkg.in/yaml.v3"
)

// RFC3339Time is a time.Time that marshals to and from RFC3339 format.
type RFC3339Time time.Time

func (t *RFC3339Time) MarshalYAML() (any, error) {
	return time.Time(*t).Format(time.RFC3339), nil
}

func (t *RFC3339Time) UnmarshalYAML(value *yaml.Node) error {
	tt, err := time.Parse(time.RFC3339, value.Value)
	if err != nil {
		return err
	}
	*t = RFC3339Time(tt)
	return nil
}

func (t RFC3339Time) String() string {
	return time.Time(t).Format(time.RFC3339)
}
