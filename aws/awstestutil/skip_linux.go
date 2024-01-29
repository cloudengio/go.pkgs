// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build linux

package awstestutil

import "testing"

func SkipAWSTests(_ *testing.T) {}

func AWSTestMain(m *testing.M, service **AWS, opts ...Option) {
	withGnomock(m, service, opts)
}
