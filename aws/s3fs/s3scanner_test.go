// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs_test

import (
	"testing"

	"cloudeng.io/aws/s3fs"
)

func TestDirectoryBucketNames(t *testing.T) {

	name := "bucket-base-name--usw2-az1--x-s3"
	if got, want := s3fs.IsDirectoryBucket(name), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := s3fs.DirectoryBucketAZ(name), "usw2-az1"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
