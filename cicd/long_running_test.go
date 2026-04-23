// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import (
	"sync"
	"testing"
)

func resetOnce() { parseOnce = sync.Once{} }

func TestParseLongRunningTestsEnv(t *testing.T) {
	tests := []struct {
		value       string
		wantEnabled bool
		wantLevel   int
		wantRegex   string // empty means neverMatch
		wantNoMatch []string
		wantMatch   []string
	}{
		{
			value:       "",
			wantEnabled: false,
			wantLevel:   0,
			wantNoMatch: []string{"anything"},
		},
		{
			value:       "1",
			wantEnabled: true,
			wantLevel:   1,
			wantNoMatch: []string{"anything"},
		},
		{
			value:       "2",
			wantEnabled: true,
			wantLevel:   2,
			wantNoMatch: []string{"anything"},
		},
		{
			value:       "Lifecycle",
			wantEnabled: true,
			wantLevel:   0,
			wantMatch:   []string{"TestVMLifecycle", "TestLifecycle"},
			wantNoMatch: []string{"TestStart", "TestStop"},
		},
		{
			value:       "Test.*VM",
			wantEnabled: true,
			wantLevel:   0,
			wantMatch:   []string{"TestStartVM", "TestStopVM"},
			wantNoMatch: []string{"TestStart", "BenchmarkVM"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			// parseLongRunningTestsEnv reads the env directly on each call;
			// the Once-cached wrapper is tested separately via LongRunnningTest.
			t.Setenv(LongRunningTestsEnv, tc.value)
			gotEnabled, gotLevel, gotRegex := parseLongRunningTestsEnv()

			if gotEnabled != tc.wantEnabled {
				t.Errorf("enabled: got %v, want %v", gotEnabled, tc.wantEnabled)
			}
			if gotLevel != tc.wantLevel {
				t.Errorf("level: got %v, want %v", gotLevel, tc.wantLevel)
			}
			for _, s := range tc.wantMatch {
				if !gotRegex.MatchString(s) {
					t.Errorf("regex %q should match %q", gotRegex, s)
				}
			}
			for _, s := range tc.wantNoMatch {
				if gotRegex.MatchString(s) {
					t.Errorf("regex %q should not match %q", gotRegex, s)
				}
			}
		})
	}
}

func TestLongRunnningTest(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		testName    string
		level       int
		wantSkipped bool
	}{
		{
			name:        "not enabled",
			envValue:    "",
			testName:    "TestFoo",
			level:       1,
			wantSkipped: true,
		},
		{
			name:        "enabled by level, test level within range",
			envValue:    "2",
			testName:    "TestFoo",
			level:       1,
			wantSkipped: false,
		},
		{
			name:        "enabled by level, test level equals limit",
			envValue:    "2",
			testName:    "TestFoo",
			level:       2,
			wantSkipped: false,
		},
		{
			name:        "enabled by level, test level exceeds limit",
			envValue:    "1",
			testName:    "TestFoo",
			level:       2,
			wantSkipped: true,
		},
		{
			name:        "enabled by regex, name matches",
			envValue:    "Lifecycle",
			testName:    "TestVMLifecycle",
			level:       0,
			wantSkipped: false,
		},
		{
			name:        "enabled by regex, name does not match",
			envValue:    "Lifecycle",
			testName:    "TestStart",
			level:       0,
			wantSkipped: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(LongRunningTestsEnv, tc.envValue)
			resetOnce()

			m := &mockT{name: tc.testName}
			LongRunnningTest(m, tc.level)
			if m.skipped != tc.wantSkipped {
				t.Errorf("skipped: got %v, want %v (skipMsg: %q)", m.skipped, tc.wantSkipped, m.skipMsg)
			}
		})
	}
}
