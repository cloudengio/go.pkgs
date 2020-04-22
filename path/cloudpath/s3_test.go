// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"testing"

	"cloudeng.io/path/cloudpath"
)

var exampleParameters = map[string][]string{
	"a": {"b"},
	"c": {"d"},
}

func TestS3(t *testing.T) {
	data := []matcherTestSpec{
		{
			"https://s3.us-west-2.amazonaws.com",
			cloudpath.AWSS3, "s3.us-west-2.amazonaws.com", "", "", '/', nil,
		},
		{
			"https://s3.us-west-2.amazonaws.com?a=b&c=d",
			cloudpath.AWSS3, "s3.us-west-2.amazonaws.com", "", "", '/', exampleParameters,
		},
		{
			"https://s3.us-west-2.amazonaws.com/",
			cloudpath.AWSS3, "s3.us-west-2.amazonaws.com", "", "/", '/', nil,
		},
		{
			"https://s3.us-west-2.amazonaws.com/mybucket/puppy.jpg",
			cloudpath.AWSS3, "s3.us-west-2.amazonaws.com", "mybucket", "/mybucket/puppy.jpg", '/', nil,
		},
		{
			"https://my.bucket.s3.us-west-2.amazonaws.com/kitten.png",
			cloudpath.AWSS3, "my.bucket.s3.us-west-2.amazonaws.com", "my.bucket", "/kitten.png", '/', nil,
		},
		{
			"s3://my-bucket/files-on-s3",
			cloudpath.AWSS3, "", "my-bucket", "/my-bucket/files-on-s3", '/', nil,
		},
		{
			"s3://",
			cloudpath.AWSS3, "", "", "", '/', nil,
		},
		{
			"s3://b",
			cloudpath.AWSS3, "", "b", "/b", '/', nil,
		},
		{
			"s3://b/",
			cloudpath.AWSS3, "", "b", "/b/", '/', nil,
		},
	}
	if err := testMatcher(cloudpath.AWSS3Matcher, data); err != nil {
		t.Errorf("%v", err)
	}

	if err := testNoMatch(cloudpath.AWSS3Matcher, []string{
		"",
		string([]byte{0x0}), // invalid URL
		"https://s.us-west-2.amazonaws.com",
		"/a/b/c",
		`c:\`,
		`\\?c:`,
		`\\host\share\a`,
		"gs:/a/b",
		"https://storage.cloud.google.com/bucket/path",
	}); err != nil {
		t.Errorf("%v", err)
	}
}
