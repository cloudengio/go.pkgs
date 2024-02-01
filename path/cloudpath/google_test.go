// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"testing"

	"cloudeng.io/path/cloudpath"
)

func TestGoogleCloudStorage(t *testing.T) {
	data := []matcherTestSpec{
		{
			"gs://",
			cloudpath.GoogleCloudStorage, "", "", "", "", "", '/', nil,
		},
		{
			"gs://bucket",
			cloudpath.GoogleCloudStorage, "", "", "bucket", "bucket", "", '/', nil,
		},
		{
			"gs://bucket/",
			cloudpath.GoogleCloudStorage, "", "", "bucket", "bucket/", "", '/', nil,
		},
		{
			"gs://bucket/object",
			cloudpath.GoogleCloudStorage, "", "", "bucket", "bucket/object", "object", '/', nil,
		},
		{
			"gs://bucket/object/",
			cloudpath.GoogleCloudStorage, "", "", "bucket", "bucket/object/", "object/", '/', nil,
		},
		{
			"https://storage.cloud.google.com/bucket/path",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "bucket", "/bucket/path", "path", '/', nil,
		},
		{
			"https://storage.cloud.google.com/bucket/path?a=b&c=d",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "bucket", "/bucket/path", "path", '/', exampleParameters,
		},
		{
			"https://storage.cloud.google.com",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "", "", "", '/', nil,
		},
		{
			"https://storage.cloud.google.com/",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "", "/", "", '/', nil,
		},
		{
			"https://storage.cloud.google.com/bucket",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "bucket", "/bucket", "", '/', nil,
		},
		{
			"https://storage.cloud.google.com/bucket/",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "bucket", "/bucket/", "", '/', nil,
		},
		{
			"https://storage.googleapis.com/storage/v1/b/bucket/path",
			cloudpath.GoogleCloudStorage, "storage.googleapis.com", "", "bucket", "/bucket/path", "path", '/', nil,
		},
		{
			"https://storage.googleapis.com/download/storage/v1/b/bucket/o/path",
			cloudpath.GoogleCloudStorage, "storage.googleapis.com", "", "bucket", "/bucket/o/path", "/path", '/', nil,
		},
		{
			"https://storage.googleapis.com/upload/storage/v1/b/bucket/o?name=/path",
			cloudpath.GoogleCloudStorage, "storage.googleapis.com", "", "bucket", "/bucket/o", "/path", '/', map[string][]string{"name": {"/path"}},
		},
		{
			"https://storage.googleapis.com/batch/storage/v1/b/",
			cloudpath.GoogleCloudStorage, "storage.googleapis.com", "", "", "/", "", '/', nil,
		},
	}
	if err := testMatcher(cloudpath.GoogleCloudStorageMatcher, data[:]); err != nil {
		t.Errorf("%v", err)
	}

	if err := testNoMatch(cloudpath.GoogleCloudStorageMatcher, []string{
		"",
		string([]byte{0x0}), // invalid URL
		"https://s.us-west-2.amazonaws.com",
		"s3:/a/b",
		"https://my.bucket.s3.us-west-2.amazonaws.com/kitten.png",
		"/a/b/c",
		`c:\`,
		`\\?c:`,
		`\\host\share\a`,
		"https://storage.googleapis.com",
		"https://storage.googleapis.com/wrong/prefix",
	}); err != nil {
		t.Errorf("%v", err)
	}
}
