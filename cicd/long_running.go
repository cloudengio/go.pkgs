// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package cicd provides support for working with CI environments.
package cicd

import (
	"fmt"
	"os"
	"regexp"
	"sync"
)

// LongRunningTestsEnv, is used to control whether and which long running tests
// are to be run.
// No long running tests are run if this variable is not set.
// If set, it may refer to either a numeric level (ie. 1, 2, etc) or a regular
// expression as per go test '-run'. The numeric level is used to control ever
// longer running tests, level 1 for < 10 minutes, level 2 for < 30 minutes, etc.
// The regular expression is used to control which tests are run based on their
// name, for example setting it to "Lifecycle" would run only tests with
// "Lifecycle" in their name.
const LongRunningTestsEnv = "CLOUDENG_LONG_RUNNING_TESTS"

var (
	parseOnce    sync.Once
	enabledOnce  bool
	levelOnce    int
	regexOnce    *regexp.Regexp
	parseErrOnce error
	neverMatch   = regexp.MustCompile("$.^") // $.^ will not match anything
)

// ParseLongRunningTestsEnv parses the CLOUDENG_LONG_RUNNING_TESTS environment variable
// and returns whether long-running tests are enabled, the numeric level if it is a
// number, and the regular expression if it is not a number. The results are cached
// after the first call.
func ParseLongRunningTestsEnv() (enabled bool, level int, regex *regexp.Regexp, err error) {
	parseOnce.Do(func() {
		enabledOnce, levelOnce, regexOnce, parseErrOnce = parseLongRunningTestsEnv()
	})
	return enabledOnce, levelOnce, regexOnce, parseErrOnce
}

func parseLongRunningTestsEnv() (enabled bool, level int, regex *regexp.Regexp, err error) {
	v := os.Getenv(LongRunningTestsEnv)
	if v == "" {
		return false, 0, neverMatch, nil
	}
	var l int
	if _, err := fmt.Sscanf(v, "%d", &l); err == nil {
		return true, l, neverMatch, nil
	}
	re, err := regexp.Compile(v)
	if err != nil {
		return false, 0, neverMatch, fmt.Errorf("invalid regular expression in %s: %w", LongRunningTestsEnv, err)
	}
	return true, 0, re, nil
}

// LongRunningTest declares the calling test as a long-running one of a given level
// that should only be run if requested via the CLOUDENG_LONG_RUNNING_TESTS environment
// variable. See the documentation for CLOUDENG_LONG_RUNNING_TESTS for details on how
// to control which long-running tests are run. In short, if not set, no
// long running tests are run; if set to a number, only long-running tests with that
// level and below are run; if set to a non-number, only long-running tests with
// names matching that regular expression are run.
func LongRunningTest(t TestingT, level int) {
	t.Helper()
	lrEnabled, lrLevel, lrRegex, err := ParseLongRunningTestsEnv()
	if err != nil {
		t.Fatalf("invalid value in %s: %v", LongRunningTestsEnv, err)
	}
	if !lrEnabled {
		t.Skipf("skipping long-running test; set %v to enable", LongRunningTestsEnv)
	}
	if lrLevel > 0 && level > lrLevel {
		t.Skipf("skipping long-running test with level %d; only level %d and below are enabled", level, lrLevel)
	}
	if lrRegex != neverMatch && !lrRegex.MatchString(t.Name()) {
		t.Skipf("skipping long-running test; test name %q does not match regex %q in %s", t.Name(), lrRegex.String(), LongRunningTestsEnv)
	}
}
