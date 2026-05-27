// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockerapi_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"cloudeng.io/vms/dockervm/dockerapi"
)

const testImage = "alpine:latest"

func TestMain(m *testing.M) {
	ctx := context.Background()

	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(os.Stderr, "docker CLI not found in PATH; skipping tests")
		os.Exit(0)
	}

	// Verify the Docker daemon is reachable.
	if out, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}").Output(); err != nil {
		fmt.Fprintf(os.Stderr, "docker info failed: %v; skipping tests\n", err)
		os.Exit(0)
	} else if strings.TrimSpace(string(out)) == "" {
		fmt.Fprintln(os.Stderr, "docker daemon not running; skipping tests")
		os.Exit(0)
	}

	// Pull the test image if not already present.
	if err := exec.CommandContext(ctx, "docker", "pull", testImage).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "docker pull %s failed: %v; skipping tests\n", testImage, err)
		os.Exit(0)
	}

	code := m.Run()

	// Clean up any leftover test containers.
	cleanupTestContainers(ctx)

	os.Exit(code)
}

// cleanupTestContainers removes any containers whose names begin with "vmstest-".
func cleanupTestContainers(ctx context.Context) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=vmstest-", "--format", "{{.Names}}").Output()
	if err != nil {
		return
	}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name = strings.TrimSpace(name)
		if name == "" || !strings.HasPrefix(name, "vmstest-") {
			continue
		}
		if info, found, err := dockerapi.InspectContainer(ctx, name); err == nil && found {
			if info.State.Running {
				exec.CommandContext(ctx, "docker", "stop", name).Run() //nolint:errcheck
			}
		}
		exec.CommandContext(ctx, "docker", "rm", "--force", name).Run() //nolint:errcheck
	}
}
