// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage

import (
	"context"
	"fmt"
)

// PullOptions are the options for a pull that every tool that can pull
// understands. A tool that understands neither ignores both.
type PullOptions struct {
	// Insecure allows the image to be pulled from a registry without
	// verifying its certificate.
	Insecure bool
	// Concurrency is the concurrency to pull with, or 0 to leave it to the
	// tool.
	Concurrency int
}

// Puller is implemented by something that can fetch an image into the local
// store of whatever creates VMs or containers from it, such as a provider
// backed by a tool with a pull command of its own.
type Puller interface {
	Pull(ctx context.Context, ref Reference, opts PullOptions) error
}

// Pull pulls ref with p, and does nothing at all for a reference that names
// no registry: there is nowhere to pull such an image from, and it is the
// local store that is expected to hold it already, see Reference.Remote.
func Pull(ctx context.Context, p Puller, ref Reference, opts PullOptions) error {
	if !ref.Remote() {
		return nil
	}
	if p == nil {
		return fmt.Errorf("ociimage: cannot pull %v: no puller", ref)
	}
	return p.Pull(ctx, ref, opts)
}
