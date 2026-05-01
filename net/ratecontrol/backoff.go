// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"time"

	"cloudeng.io/algo/ratecontrol"
)

type Backoff ratecontrol.Backoff

// NewExpontentialBackoff returns a instance of Backoff that implements
// an exponential backoff algorithm starting with the specified initial
// delay and continuing for the specified number of steps.
func NewExpontentialBackoff(initial time.Duration, steps int) ratecontrol.Backoff {
	return ratecontrol.NewExponentialBackoff(initial, steps)
}
