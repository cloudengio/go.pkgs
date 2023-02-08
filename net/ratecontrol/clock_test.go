// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ratecontrol provides mechanisms for controlling the rate
// at which requests are made.
package ratecontrol

import "time"

type TestClock struct {
	TickDurationValue time.Duration
	AfterValue        time.Duration
	Called            int
	AfterDurations    []time.Duration
}

func (c *TestClock) Tick() int {
	c.Called++
	return time.Now().Minute()
}

func (c *TestClock) TickDuration() time.Duration {
	c.Called++
	return c.TickDurationValue
}

func (c *TestClock) after(d time.Duration) <-chan time.Time {
	c.Called++
	c.AfterDurations = append(c.AfterDurations, d)
	return time.After(c.AfterValue)
}
