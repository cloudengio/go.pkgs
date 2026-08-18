// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
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
// ExponentialBackoff otherwise. If either IntialDelay and Steps are less than
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

// RateConfig represents the configuration for a rate controller.
type RateConfig struct {
	Tick            time.Duration `yaml:"tick" doc:"the duration of a tick"`
	RequestsPerTick int           `yaml:"requests_per_tick" doc:"the number of requests per tick"`
	BytesPerTick    int           `yaml:"bytes_per_tick" doc:"the number of bytes per tick"`
}

// RateControlConfig combines a rate with an exponential backoff.
type RateControlConfig struct {
	Rate               RateConfig               `yaml:"rate_control" doc:"the rate control parameters"`
	ExponentialBackoff ExponentialBackoffConfig `yaml:"exponential_backoff" doc:"the exponential backoff parameters"`
}

// NewController creates a new Controller based on the configuration.
func (rc RateControlConfig) NewController() *Controller {
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
