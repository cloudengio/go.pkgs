// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd_test

import (
	"fmt"
	"regexp"
	"testing"
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

func TestConfigManager_FirstMatch(t *testing.T) {
	var mgr cicd.ConfigManager[int]
	mgr.SetDefault(-1)

	mgr.Set(regexp.MustCompile(`foo.*`), 1)
	mgr.Set(regexp.MustCompile(`foo_bar`), 2)
	mgr.Set(regexp.MustCompile(`.*bar`), 3)

	// Matches both foo.* and foo_bar, and .*bar, but foo.* is added first.
	if got := mgr.Get("foo_bar"); got != 1 {
		t.Errorf("got %v, want 1", got)
	}

	// Now check if a new manager resolves .*bar first when order is changed.
	var mgr2 cicd.ConfigManager[int]
	mgr2.SetDefault(-1)
	mgr2.Set(regexp.MustCompile(`.*bar`), 3)
	mgr2.Set(regexp.MustCompile(`foo.*`), 1)
	mgr2.Set(regexp.MustCompile(`foo_bar`), 2)

	if got := mgr2.Get("foo_bar"); got != 3 {
		t.Errorf("got %v, want 3", got)
	}
}
