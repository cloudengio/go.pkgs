// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage

import "fmt"

// Config configures the image a backend creates its VMs or containers from,
// and how that image is obtained. It carries serialisation tags so that a
// backend can embed it directly in its own configuration, rather than
// restating these, and so that every backend spells them the same way.
type Config struct {
	Image           string `yaml:"image" doc:"the OCI image reference to create VMs from, such as 'linux-ci', 'linux-ci:latest' or 'ghcr.io/cirruslabs/ubuntu:latest'."`
	PullRemoteOnRun bool   `yaml:"pull_remote_on_run" doc:"pull the image from its registry before creating each VM, so that a moving tag such as :latest is picked up without restarting. Ignored for a reference that names no registry, which has nothing to pull from."`
	Insecure        bool   `yaml:"pull_insecure" doc:"allow pulling the image from a registry without verifying its certificate."`
	Concurrency     int    `yaml:"pull_concurrency" doc:"the concurrency to pull the image with; 0 leaves it to the tool that pulls it."`
}

// Reference parses the configured image.
func (c Config) Reference() (Reference, error) {
	return Parse(c.Image)
}

// Validate reports a configuration that cannot be used: one that names no
// image, or one whose image is not a reference. It is for a backend to call
// when its own configuration is read, so that a mistake in a configuration
// file is reported then rather than when the first VM fails to be created.
func (c Config) Validate() error {
	if c.Image == "" {
		return fmt.Errorf("image is required")
	}
	if _, err := c.Reference(); err != nil {
		return err
	}
	return nil
}

// ShouldPull reports whether ref should be pulled before it is used: the
// configuration asks for it and the reference names a registry to pull from.
// A local reference has nothing to pull from, so asking for it is not an
// error, it simply does not apply.
func (c Config) ShouldPull(ref Reference) bool {
	return c.PullRemoteOnRun && ref.Remote()
}

// PullOptions returns the configured options for a pull.
func (c Config) PullOptions() PullOptions {
	return PullOptions{Insecure: c.Insecure, Concurrency: c.Concurrency}
}
