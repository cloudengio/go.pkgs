// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage_test

import (
	"errors"
	"testing"

	"cloudeng.io/vms/ociimage"
)

// TestSourceValidate covers the mistakes a configuration file can make that
// are worth reporting when it is read, rather than when the first VM fails to
// be created.
func TestSourceValidate(t *testing.T) {
	if err := (ociimage.Source{Image: "ghcr.io/cirruslabs/ubuntu:latest"}).Validate(); err != nil {
		t.Errorf("a valid configuration was rejected: %v", err)
	}
	// No image at all.
	err := (ociimage.Source{}).Validate()
	if !errors.Is(err, ociimage.ErrInvalidReference) {
		t.Errorf("got %v, want ErrInvalidReference", err)
	}
	// An image that is not a reference, which Validate reports as such.
	err = (ociimage.Source{Image: "ghcr.io//ubuntu"}).Validate()
	if !errors.Is(err, ociimage.ErrInvalidReference) {
		t.Errorf("got %v, want ErrInvalidReference", err)
	}
}

func TestSourceReference(t *testing.T) {
	cfg := ociimage.Source{Image: "mm-1:5001/linux-ci:latest"}
	ref, err := cfg.Reference()
	if err != nil {
		t.Fatalf("Reference: %v", err)
	}
	if got, want := ref.Name, "linux-ci"; got != want {
		t.Errorf("Name: got %q, want %q", got, want)
	}
	if got, want := ref.String(), cfg.Image; got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
	if _, err := (ociimage.Source{Image: "linux-ci:"}).Reference(); !errors.Is(err, ociimage.ErrInvalidReference) {
		t.Errorf("got %v, want ErrInvalidReference", err)
	}
}

// TestSourceShouldPull covers the pull policy: a configuration that asks for
// a pull gets one only for an image that has a registry to be pulled from.
func TestSourceShouldPull(t *testing.T) {
	for _, tc := range []struct {
		image string
		ask   bool
		want  bool
	}{
		// Asked for, and there is somewhere to pull from.
		{"ghcr.io/cirruslabs/ubuntu:latest", true, true},
		{"mm-1:5001/linux-ci", true, true},
		{"linux-ci@sha256:0123456789abcdef", true, true},
		// Asked for, but the image is local: nothing to pull from, which is
		// not an error, it simply does not apply.
		{"linux-ci", true, false},
		{"linux-ci:latest", true, false},
		{"myorg/linux-ci", true, false},
		// Not asked for.
		{"ghcr.io/cirruslabs/ubuntu:latest", false, false},
		{"linux-ci", false, false},
	} {
		cfg := ociimage.Source{Image: tc.image, PullRemoteOnRun: tc.ask}
		ref, err := cfg.Reference()
		if err != nil {
			t.Errorf("Reference(%q): %v", tc.image, err)
			continue
		}
		if got := cfg.ShouldPull(ref); got != tc.want {
			t.Errorf("%q with pull_remote_on_run=%v: ShouldPull = %v, want %v", tc.image, tc.ask, got, tc.want)
		}
	}
}

func TestSourcePullOptions(t *testing.T) {
	cfg := ociimage.Source{Image: "linux-ci", Insecure: true, Concurrency: 4}
	got := ociimage.ResolvePullOptions(cfg.PullOptions()...)
	if want := (ociimage.PullSettings{Insecure: true, Concurrency: 4}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	// An unconfigured pull leaves both to the tool that performs it.
	got = ociimage.ResolvePullOptions((ociimage.Source{}).PullOptions()...)
	if want := (ociimage.PullSettings{}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
