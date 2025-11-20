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

// FlexTime is a time.Time that can be unmarshaled from time.RFC3339,
// time.DateTime, time.TimeOnly or time.DateOnly formats. It is always
// marshaled to time.RFC3339.
type FlexTime time.Time

func (t *FlexTime) MarshalYAML() (any, error) {
	return time.Time(*t).Format(time.RFC3339), nil
}

func (t *FlexTime) UnmarshalYAML(value *yaml.Node) error {
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		tt, err := time.Parse(format, value.Value)
		if err == nil {
			*t = FlexTime(tt)
			return nil
		}
	}
	return nil
}

func (t FlexTime) String() string {
	return time.Time(t).Format(time.RFC3339)
}
