// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags

import (
	"fmt"
	"time"
)

// Time represents a time.Time that can be used as a flag.Value. The time
// can be expressed in time.RFC3339, time.DateTime, time.TimeOnly or time.DateOnly
// formats.
type Time struct {
	opt   string
	value time.Time
	set   bool
}

// Set implements flag.Value.
func (tf *Time) Set(v string) error {
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		if t, err := time.Parse(format, v); err == nil {
			tf.opt = v
			tf.value = t
			tf.set = true
			return nil
		}
	}
	return fmt.Errorf("invalid time: %v, use one of RFC3339, Date and Time, Date or Time only formats", v)
}

// String implements flag.Value.
func (tf *Time) String() string {
	return tf.opt
}

// Value implements flag.Getter.
func (tf *Time) Get() interface{} {
	return tf.value
}

// IsSet returns true if the value has been set.
func (tf *Time) IsDefault() bool {
	return !tf.set
}
