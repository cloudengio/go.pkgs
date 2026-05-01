// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol

import (
	"cloudeng.io/algo/ratecontrol"
)

type Limiter ratecontrol.Limiter

// Controller implements Limiter and is used to control the rate at which
// requests are made and to implement backoff when the remote server is
// unwilling to process a request. Controller is safe to use concurrently.
type Controller struct {
	*ratecontrol.Controller
}

// New returns a new Controller configuring using the specified options.
func New(opts ...Option) *Controller {
	ropts := make([]ratecontrol.Option, len(opts))
	for i, opt := range opts {
		ropts[i] = ratecontrol.Option(opt)
	}
	return &Controller{ratecontrol.New(ropts...)}
}
