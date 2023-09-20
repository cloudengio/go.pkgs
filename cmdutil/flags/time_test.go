// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func TestTimeFlag(t *testing.T) {
	tf := &flags.Time{}
	for i, tc := range []string{
		"2021-10-10",                // date only
		"2021-10-10T03:03:03-07:00", // RFC 3339
		"03:03:05",                  // time only
		"2021-10-10 03:03:05",       // date time
	} {
		if err := tf.Set(tc); err != nil {
			t.Errorf("%v: %v", i, err)
		}
	}
}
