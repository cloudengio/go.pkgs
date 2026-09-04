// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
)

// Time represents a time.Time that can be used as a flag.Value. It accepts
// the same formats as cmdtypes.FlexTime, ie. time.RFC3339, time.DateTime,
// time.TimeOnly or time.DateOnly, and is parsed by it, so that a value given
// on the command line is read the same way as one read from JSON or YAML.
type Time struct {
	opt   string
	value cmdtypes.FlexTime
	set   bool
}

// Set implements flag.Value.
func (tf *Time) Set(v string) error {
	parsed, err := cmdtypes.ParseFlexTime(v)
	if err != nil {
		return err
	}
	tf.opt = v
	tf.value = parsed
	tf.set = true
	return nil
}

// String implements flag.Value.
func (tf *Time) String() string {
	return tf.opt
}

// Get implements flag.Getter, returning the time.Time represented by the flag.
func (tf *Time) Get() any {
	return tf.value.Time()
}

// Time returns the time.Time represented by the flag.
func (tf *Time) Time() time.Time {
	return tf.value.Time()
}

// FlexTime returns the value as a cmdtypes.FlexTime, for use where the same
// value is also read from or written to JSON or YAML.
func (tf *Time) FlexTime() cmdtypes.FlexTime {
	return tf.value
}

// IsSet returns true if the value has been set.
func (tf *Time) IsDefault() bool {
	return !tf.set
}
