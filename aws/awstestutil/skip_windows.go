// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package awstestutil

import (
	"os"
	"testing"
)

func SkipAWSTests(t *testing.T) {
	t.Skip("skipping test on windows")
}

func AWSTestMain(m *testing.M, service **AWS, opts ...Option) {
	os.Exit(m.Run())
}
