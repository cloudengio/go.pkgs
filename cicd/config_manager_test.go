// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd_test

import (
	"fmt"
	"regexp"
	"time"

	"cloudeng.io/cicd"
)

// ExampleConfigManager demonstrates centralizing per-implementation test
// configuration. A shared test suite calls Get with the running test name
// to obtain the parameters that match that implementation; names that
// match no regex fall back to the default.
func ExampleConfigManager() {
	type VMConfig struct {
		Image   string
		Timeout time.Duration
	}

	var mgr cicd.ConfigManager[VMConfig]
	mgr.SetDefault(VMConfig{Image: "linux:latest", Timeout: 5 * time.Minute})
	mgr.Set(regexp.MustCompile(`MacOS`), VMConfig{Image: "macos:latest", Timeout: 15 * time.Minute})
	mgr.Set(regexp.MustCompile(`Windows`), VMConfig{Image: "windows:latest", Timeout: 20 * time.Minute})

	for _, testName := range []string{
		"TestProvisionMacOS",
		"TestProvisionWindows",
		"TestProvisionLinux",
	} {
		cfg := mgr.Get(testName)
		fmt.Printf("%-26s image=%-18s timeout=%v\n", testName, cfg.Image, cfg.Timeout)
	}
	// Output:
	// TestProvisionMacOS         image=macos:latest       timeout=15m0s
	// TestProvisionWindows       image=windows:latest     timeout=20m0s
	// TestProvisionLinux         image=linux:latest       timeout=5m0s
}
