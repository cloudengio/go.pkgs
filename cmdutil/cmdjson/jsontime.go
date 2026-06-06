// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdjson

import (
	"encoding/json"
	"time"
)

// RFC3339Time is a time.Time that marshals to and from RFC3339 format.
type RFC3339Time time.Time

func (t *RFC3339Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(*t).Format(time.RFC3339))
}

func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tt, err := time.Parse(time.RFC3339, s)
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

func (t FlexTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

func (t *FlexTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		tt, err := time.Parse(format, s)
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
