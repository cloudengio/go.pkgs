// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ExponentialBackoffConfig represents a configuration struct that can be used
// to create a Backoff instance that implements an exponential backoff.
type ExponentialBackoffConfig struct {
	InitialDelay   time.Duration `yaml:"initial_delay" doc:"the initial delay between retries for exponential backoff"`
	Steps          int           `yaml:"steps" doc:"the number of steps of exponential backoff before giving up"`
	RandomizeStart bool          `yaml:"randomize_start" doc:"if true, a random offset of up to initial_delay will be used to randomize the start of the backoff period to avoid thundering herd issues when many retries are attempted at the same time."`
}

// NewBackoff creates a ExponentialBackoffOffset if RandomizeStart is set, and
// ExponentialBackoff otherwise. If either InitialDelay or Steps are less than
// or equal to zero then NoBackoff is returned.
func (ebc ExponentialBackoffConfig) NewBackoff() Backoff {
	if ebc.InitialDelay > 0 && ebc.Steps > 0 {
		if ebc.RandomizeStart {
			return NewExponentialBackoffOffset(ebc.InitialDelay, ebc.Steps)
		}
		return NewExponentialBackoff(ebc.InitialDelay, ebc.Steps)
	}
	return NoBackoff{}
}

// BackoffOption returns an Option representing the backoff configuration that
// can be used when creating a Controller.
func (ebc ExponentialBackoffConfig) BackoffOption() Option {
	return WithExponentialBackoff(ebc.InitialDelay, ebc.Steps, ebc.RandomizeStart)
}

// TotalTimeout returns the total time that a backoff created from this
// configuration can spend waiting across all of its steps, ie.
// InitialDelay * (2^Steps - 1). It does not account for any randomization
// of the start of the backoff period (RandomizeStart). Zero is returned
// for a configuration that yields NoBackoff.
func (ebc ExponentialBackoffConfig) TotalTimeout() time.Duration {
	if ebc.InitialDelay <= 0 || ebc.Steps <= 0 {
		return 0
	}
	total, delay := time.Duration(0), ebc.InitialDelay
	for i := 0; i < ebc.Steps; i++ {
		total += delay
		if total < 0 {
			// Overflowed; return the largest possible timeout.
			return math.MaxInt64
		}
		delay *= 2
		if delay < 0 {
			delay = math.MaxInt64
		}
	}
	return total
}

// String returns a compact representation of the backoff configuration and
// the total time it can spend waiting, eg. "initial=100ms steps=10 total=1m42.3s"
// with " randomized" appended if RandomizeStart is set. A configuration that
// yields NoBackoff is represented as "no backoff".
func (ebc ExponentialBackoffConfig) String() string {
	if ebc.InitialDelay <= 0 || ebc.Steps <= 0 {
		return "no backoff"
	}
	randomized := ""
	if ebc.RandomizeStart {
		randomized = " randomized"
	}
	return fmt.Sprintf("initial=%v steps=%v total=%v%s", ebc.InitialDelay, ebc.Steps, ebc.TotalTimeout(), randomized)
}

// RateConfig represents the configuration for a rate controller.
type RateConfig struct {
	Tick            time.Duration `yaml:"tick" doc:"the duration of a tick"`
	RequestsPerTick int           `yaml:"requests_per_tick" doc:"the number of requests per tick"`
	BytesPerTick    int           `yaml:"bytes_per_tick" doc:"the number of bytes per tick"`
}

// String returns a compact representation of the rate configuration,
// eg. "10 requests/1s, 1000 bytes/1s". Limits that are not configured are
// omitted and "no rate limits" is returned when neither is configured.
func (rc RateConfig) String() string {
	parts := make([]string, 0, 2)
	if rc.RequestsPerTick > 0 {
		parts = append(parts, fmt.Sprintf("%v requests/%v", rc.RequestsPerTick, rc.Tick))
	}
	if rc.BytesPerTick > 0 {
		parts = append(parts, fmt.Sprintf("%v bytes/%v", rc.BytesPerTick, rc.Tick))
	}
	if len(parts) == 0 {
		return "no rate limits"
	}
	return strings.Join(parts, ", ")
}

// ControllerConfig combines a rate with an exponential backoff.
type ControllerConfig struct {
	Rate               RateConfig               `yaml:"rate_control" doc:"the rate control parameters"`
	ExponentialBackoff ExponentialBackoffConfig `yaml:"exponential_backoff" doc:"the exponential backoff parameters"`
}

// NewController creates a new Controller based on the configuration.
func (rc ControllerConfig) NewController() *Controller {
	opts := []Option{}
	if rc.Rate.BytesPerTick > 0 {
		opts = append(opts, WithBytesPerTick(rc.Rate.Tick, rc.Rate.BytesPerTick))
	}
	if rc.Rate.RequestsPerTick > 0 {
		opts = append(opts, WithRequestsPerTick(rc.Rate.Tick, rc.Rate.RequestsPerTick))
	}
	if rc.ExponentialBackoff.InitialDelay > 0 {
		opts = append(opts, rc.ExponentialBackoff.BackoffOption())
	}
	return New(opts...)
}
