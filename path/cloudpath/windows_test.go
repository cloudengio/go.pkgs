// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"testing"

	"cloudeng.io/path/cloudpath"
)

func TestWindows(t *testing.T) {
	data := []matcherTestSpec{
		{
			"c:",
			cloudpath.WindowsFileSystem, "localhost", "c", "c:", '\\', nil,
		},
		{
			`c:\`,
			cloudpath.WindowsFileSystem, "localhost", "c", `c:\`, '\\', nil,
		},
		{
			`c:\a\b`,
			cloudpath.WindowsFileSystem, "localhost", "c", `c:\a\b`, '\\', nil,
		},
		{
			`Z:a\b`,
			cloudpath.WindowsFileSystem, "localhost", "Z", `Z:a\b`, '\\', nil,
		},
		{
			`\\host`,
			cloudpath.WindowsFileSystem, "host", "", "", '\\', nil,
		},
		{
			`\\host\`,
			cloudpath.WindowsFileSystem, "host", "", "", '\\', nil,
		},
		{
			`\\host\server`,
			cloudpath.WindowsFileSystem, "host", "server", "", '\\', nil,
		},
		{
			`\\host\server\`,
			cloudpath.WindowsFileSystem, "host", "server", "", '\\', nil,
		},
		{
			`\\host\server\a\b`,
			cloudpath.WindowsFileSystem, "host", "server", `\a\b`, '\\', nil,
		},
		{
			`\\?c:\a\b`,
			cloudpath.WindowsFileSystem, "localhost", "c", `c:\a\b`, '\\', nil,
		},
		{
			`\\?Z:a\b`,
			cloudpath.WindowsFileSystem, "localhost", "Z", `Z:a\b`, '\\', nil,
		},
		{
			"file:///c:/a/b/c/",
			cloudpath.WindowsFileSystem, "localhost", "c", "c:/a/b/c/", '/', nil,
		},
		{
			`file://ignored/c:/a/b/c/`,
			cloudpath.WindowsFileSystem, "localhost", "c", "c:/a/b/c/", '/', nil,
		},
	}
	if err := testMatcher(cloudpath.WindowsMatcher, data); err != nil {
		t.Errorf("%v", err)
	}
	if err := testNoMatch(cloudpath.WindowsMatcher, []string{
		"",
		string([]byte{0x0}), // invalid URL
		"https://s.us-west-2.amazonaws.com",
		"s3:/a/b",
		"https://my.bucket.s3.us-west-2.amazonaws.com/kitten.png",
		"gs://bucket/object",
		"https://storage.cloud.google.com/bucket/",
		"/a/b/c",
	}); err != nil {
		t.Errorf("%v", err)
	}
}
