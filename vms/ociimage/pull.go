// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage

import (
	"context"
	"fmt"
)

// pullOptions is what the PullOption values given for a pull are applied to.
// Both it and its contents are private, so that the options for a pull can be
// given in one way only: as PullOption values.
type pullOptions struct {
	settings PullSettings
}

// PullOption sets one of the options for a pull, see Pull and Puller.
type PullOption func(*pullOptions)

// WithPullInsecure allows the image to be pulled from a registry without
// verifying its certificate, which is for a private registry whose
// certificate cannot be verified, or which serves plain HTTP.
func WithPullInsecure(insecure bool) PullOption {
	return func(o *pullOptions) { o.settings.Insecure = insecure }
}

// WithPullConcurrency sets the concurrency to pull with, leaving it to the
// tool for anything less than one.
func WithPullConcurrency(concurrency int) PullOption {
	if concurrency < 1 {
		concurrency = 0
	}
	return func(o *pullOptions) { o.settings.Concurrency = concurrency }
}

// PullSettings is what a pull was asked for: the options that every tool
// which can pull understands, and a tool that understands none of them
// ignores them all. It is how a Puller reads the options it is given, see
// ResolvePullOptions, and is only ever read: a pull is asked for with
// PullOption values.
type PullSettings struct {
	// Insecure allows the image to be pulled from a registry without
	// verifying its certificate.
	Insecure bool
	// Concurrency is the concurrency to pull with, or 0 to leave it to the
	// tool.
	Concurrency int
}

// ResolvePullOptions resolves opts, which is how a Puller reads what the pull
// it is carrying out was asked for.
func ResolvePullOptions(opts ...PullOption) PullSettings {
	var o pullOptions
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	return o.settings
}

// Puller is implemented by something that can fetch an image into the local
// store of whatever creates VMs or containers from it, such as a provider
// backed by a tool with a pull command of its own. An implementation reads the
// options it is given with ResolvePullOptions.
type Puller interface {
	Pull(ctx context.Context, ref Reference, opts ...PullOption) error
}

// Pull pulls ref with p, and does nothing at all for a reference that names
// no registry: there is nowhere to pull such an image from, and it is the
// local store that is expected to hold it already, see Reference.Remote.
func Pull(ctx context.Context, p Puller, ref Reference, opts ...PullOption) error {
	if !ref.Remote() {
		return nil
	}
	if p == nil {
		return fmt.Errorf("ociimage: cannot pull %v: no puller", ref)
	}
	return p.Pull(ctx, ref, opts...)
}
