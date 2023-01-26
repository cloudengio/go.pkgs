// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"testing"

	"cloudeng.io/path/cloudpath"
)

func TestURL(t *testing.T) {
	data := []matcherTestSpec{
		{
			"https://s3.us-west-2.amazonaws.com",
			cloudpath.HTTPS, "s3.us-west-2.amazonaws.com", "", "", "", "", '/', nil,
		},
		{
			"https://yahoo.com/a/b?a=b",
			cloudpath.HTTPS, "yahoo.com", "", "", "/a/b", "/a/b", '/', map[string][]string{"a": {"b"}},
		},
	}
	if err := testMatcher(cloudpath.URLMatcher, data); err != nil {
		t.Errorf("%v", err)
	}

	if err := testNoMatch(cloudpath.URLMatcher, []string{
		"",
		string([]byte{0x0}), // invalid URL
		"s3://a/b/",
		"/a/b/c",
		`c:\`,
		`\\?c:`,
		`\\host\share\a`,
		"gs:/a/b",
	}); err != nil {
		t.Errorf("%v", err)
	}
}

func TestURLS3(t *testing.T) {
	// Note, that with the default matchers, s3 is processed before http so URLS
	// for s3 get matched as s3 and not as https.
	data := []matcherTestSpec{
		{
			"https://s3.us-west-2.amazonaws.com",
			cloudpath.AWSS3, "s3.us-west-2.amazonaws.com", "us-west-2", "", "", "", '/', nil,
		},
	}
	if err := testMatcherSpec(cloudpath.DefaultMatchers, data); err != nil {
		t.Errorf("%v", err)
	}
}
