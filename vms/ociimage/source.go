// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage

import "fmt"

// Source is where a backend's image comes from: which image it creates its
// VMs or containers from, and how that image is obtained. It carries
// serialisation tags so that a backend can embed it inline in its own
// configuration, rather than restating these, and so that every backend
// spells them the same way.
//
// It is named for what it describes rather than being another Config, so that
// it can be embedded alongside the configuration of the tool that uses it: a
// struct cannot embed both tartvm.Config and an ociimage.Config, since the
// two would contribute the same field name.
type Source struct {
	Image           string `yaml:"image" doc:"the OCI image reference to create VMs from, such as 'linux-ci', 'linux-ci:latest' or 'ghcr.io/cirruslabs/ubuntu:latest'."`
	PullRemoteOnRun bool   `yaml:"pull_remote_on_run" doc:"pull the image from its registry before creating each VM, so that a moving tag such as :latest is picked up without restarting. Ignored for a reference that names no registry, which has nothing to pull from."`
	Insecure        bool   `yaml:"pull_insecure" doc:"allow pulling the image from a registry without verifying its certificate."`
	Concurrency     int    `yaml:"pull_concurrency" doc:"the concurrency to pull the image with; 0 leaves it to the tool that pulls it."`
	// Verify, if set, is the policy that the image's signature must satisfy,
	// see Verify and PullVerified.
	Verify *Policy `yaml:"verify" doc:"require the image to carry a signature that satisfies this policy before it is used."`
}

// Reference parses the configured image.
func (s Source) Reference() (Reference, error) {
	return Parse(s.Image)
}

// Validate reports a configuration that cannot be used: one that names no
// image, or one whose image is not a reference. It is for a backend to call
// when its own configuration is read, so that a mistake in a configuration
// file is reported then rather than when the first VM fails to be created.
// A backend that locates its own settings in its messages reports these
// itself instead, see Reference.
func (s Source) Validate() error {
	if s.Image == "" {
		return fmt.Errorf("%w: image is required", ErrInvalidReference)
	}
	ref, err := s.Reference()
	if err != nil {
		return err
	}
	if s.Verify != nil {
		if err := s.Verify.Validate(); err != nil {
			return fmt.Errorf("verify: %w", err)
		}
		if !ref.Remote() {
			return fmt.Errorf("verify: %v names no registry, so it has no signature to verify", ref)
		}
	}
	return nil
}

// ShouldPull reports whether ref should be pulled before it is used: the
// configuration asks for it and the reference names a registry to pull from.
// A local reference has nothing to pull from, so asking for it is not an
// error, it simply does not apply.
func (s Source) ShouldPull(ref Reference) bool {
	return s.PullRemoteOnRun && ref.Remote()
}

// PullOptions returns the configured options for a pull.
func (s Source) PullOptions() PullOptions {
	return PullOptions{Insecure: s.Insecure, Concurrency: s.Concurrency}
}
