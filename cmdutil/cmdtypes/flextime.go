// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes

import (
	"fmt"
	"time"
)

// FlexTimeFormats are the formats that a FlexTime is decoded from, tried in
// this order.
var FlexTimeFormats = []string{
	time.RFC3339,
	time.DateTime,
	time.TimeOnly,
	time.DateOnly,
}

// FlexTime is a time.Time that can be decoded from any of FlexTimeFormats and
// is always encoded as time.RFC3339.
//
// Encoding and decoding is implemented by MarshalText and UnmarshalText, which
// encoding/json, encoding/json/v2 and gopkg.in/yaml.v3 all use, so a single
// type serves all three without this package depending on any of them.
type FlexTime time.Time

// ParseFlexTime parses s as any of FlexTimeFormats.
func ParseFlexTime(s string) (FlexTime, error) {
	for _, format := range FlexTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return FlexTime(t), nil
		}
	}
	return FlexTime{}, fmt.Errorf("invalid time: %v, use one of time.RFC3339, time.DateTime, time.Date or time.Time only formats", s)
}

// Time returns the time.Time represented by t.
func (t FlexTime) Time() time.Time {
	return time.Time(t)
}

// String returns the time in time.RFC3339 format.
func (t FlexTime) String() string {
	return time.Time(t).Format(time.RFC3339)
}

// MarshalText implements encoding.TextMarshaler, and with it the encoding used
// for JSON and YAML.
func (t FlexTime) MarshalText() ([]byte, error) {
	return []byte(time.Time(t).Format(time.RFC3339)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, and with it the decoding
// used for JSON and YAML.
func (t *FlexTime) UnmarshalText(text []byte) error {
	parsed, err := ParseFlexTime(string(text))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
