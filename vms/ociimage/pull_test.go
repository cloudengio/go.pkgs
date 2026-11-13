// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage_test

import (
	"context"
	"errors"
	"testing"

	"cloudeng.io/vms/ociimage"
)

// recordingPuller stands in for a tool with a pull command of its own,
// recording what it was asked to pull.
type recordingPuller struct {
	refs []ociimage.Reference
	opts []ociimage.PullSettings
	err  error
}

func (p *recordingPuller) Pull(_ context.Context, ref ociimage.Reference, opts ...ociimage.PullOption) error {
	p.refs = append(p.refs, ref)
	p.opts = append(p.opts, ociimage.ResolvePullOptions(opts...))
	return p.err
}

var _ ociimage.Puller = (*recordingPuller)(nil)

func mustParse(t *testing.T, s string) ociimage.Reference {
	t.Helper()
	ref, err := ociimage.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return ref
}

// TestPull covers pulling an image that names a registry: the reference and
// the options reach the puller as they were given.
func TestPull(t *testing.T) {
	ctx := context.Background()
	p := &recordingPuller{}
	ref := mustParse(t, "ghcr.io/cirruslabs/ubuntu:latest")
	err := ociimage.Pull(ctx, p, ref,
		ociimage.WithPullInsecure(true), ociimage.WithPullConcurrency(4))
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(p.refs) != 1 || p.refs[0] != ref {
		t.Errorf("pulled %v, want [%v]", p.refs, ref)
	}
	want := ociimage.PullSettings{Insecure: true, Concurrency: 4}
	if len(p.opts) != 1 || p.opts[0] != want {
		t.Errorf("pulled with %+v, want [%+v]", p.opts, want)
	}
}

// TestPullOptions covers resolving the options of a pull: nothing is asked of
// the tool by default, and the last value given for an option is the one that
// applies, so that a caller can override what it was handed.
func TestPullOptions(t *testing.T) {
	if got, want := ociimage.ResolvePullOptions(nil), (ociimage.PullSettings{}); got != want {
		t.Errorf("by default: got %+v, want %+v", got, want)
	}
	got := ociimage.ResolvePullOptions(
		ociimage.WithPullInsecure(true),
		ociimage.WithPullConcurrency(4),
		nil,
		ociimage.WithPullConcurrency(8))
	if want := (ociimage.PullSettings{Insecure: true, Concurrency: 8}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got, want := ociimage.ResolvePullOptions(ociimage.WithPullConcurrency(4), ociimage.WithPullConcurrency(-1)), (ociimage.PullSettings{}); got != want {
		t.Errorf("negative concurrency: got %+v, want %+v", got, want)
	}
}

// TestPullLocalImage covers an image that names no registry: there is nowhere
// to pull it from, so nothing is attempted and that is not an error.
func TestPullLocalImage(t *testing.T) {
	ctx := context.Background()
	for _, image := range []string{"linux-ci", "linux-ci:latest", "myorg/linux-ci"} {
		p := &recordingPuller{err: errors.New("the puller must not be called")}
		if err := ociimage.Pull(ctx, p, mustParse(t, image)); err != nil {
			t.Errorf("Pull(%q): %v", image, err)
		}
		if len(p.refs) != 0 {
			t.Errorf("Pull(%q) attempted to pull %v", image, p.refs)
		}
	}
	// A local image is not pulled whether there is a puller or not, so the
	// absence of one does not matter.
	if err := ociimage.Pull(ctx, nil, mustParse(t, "linux-ci")); err != nil {
		t.Errorf("Pull with no puller: %v", err)
	}
}

// TestPullError covers the puller's own failure reaching the caller: the
// point of pulling is that the VM runs the current image, so a failure must
// not be mistaken for a pull that was not needed.
func TestPullError(t *testing.T) {
	failed := errors.New("the registry is unreachable")
	p := &recordingPuller{err: failed}
	err := ociimage.Pull(context.Background(), p, mustParse(t, "ghcr.io/cirruslabs/ubuntu"))
	if !errors.Is(err, failed) {
		t.Errorf("got %v, want %v", err, failed)
	}
}

// TestPullNoPuller covers an image that must be pulled with nothing to pull
// it, which is reported rather than passed over: the image would otherwise
// be used stale, or not be there at all.
func TestPullNoPuller(t *testing.T) {
	err := ociimage.Pull(context.Background(), nil, mustParse(t, "ghcr.io/cirruslabs/ubuntu"))
	if err == nil {
		t.Error("pulling a remote image with no puller was accepted")
	}
}
