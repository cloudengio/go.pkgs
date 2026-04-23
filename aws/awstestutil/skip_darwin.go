// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package awstestutil

import (
	"os"
	"testing"

	"cloudeng.io/cicd"
)

func SkipAWSTests(t *testing.T) {
	if cicd.IsGitHubActions() {
		t.Skip("skipping test on github actions")
	}
}

func AWSTestMain(m *testing.M, service **AWS, opts ...Option) {
	if cicd.IsGitHubActions() {
		os.Exit(m.Run())
	}
	withGnomock(m, service, opts)
}
