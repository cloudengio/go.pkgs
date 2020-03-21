// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath_test

import (
	"path/filepath"
	"testing"
	"unicode/utf8"

	"cloudeng.io/path/cloudpath"
)

const sep = string(filepath.Separator)

func sliceFromComponents(components ...string) []string {
	return components
}

func joinComponents(components ...string) string {
	out := ""
	for _, c := range components {
		out += c
	}
	return out
}

func TestMatch(t *testing.T) {
	data := []matcherTestSpec{
		{
			"https://s3.us-west-2.amazonaws.com",
			cloudpath.AWSS3, "s3.us-west-2.amazonaws.com", "", "", '/', nil,
		},
		{
			"s3://my-bucket/files-on-s3",
			cloudpath.AWSS3, "", "my-bucket", "/my-bucket/files-on-s3", '/', nil,
		},
		{
			"/a/b",
			cloudpath.UnixFileSystem, "localhost", "", "/a/b", '/', nil,
		},
		{
			`\\?Z:a\b`,
			cloudpath.WindowsFileSystem, "localhost", "Z", `Z:a\b`, '\\', nil,
		},
		{
			`Z:a\b`,
			cloudpath.WindowsFileSystem, "localhost", "Z", `Z:a\b`, '\\', nil,
		},
		{
			"https://storage.cloud.google.com/",
			cloudpath.GoogleCloudStorage, "storage.cloud.google.com", "", "/", '/', nil,
		},
		{
			"gs://bucket/object",
			cloudpath.GoogleCloudStorage, "", "bucket", "/bucket/object", '/', nil,
		},
		{
			"file:///a/b/c/",
			cloudpath.UnixFileSystem, "localhost", "", "/a/b/c/", '/', nil,
		},
		{
			"file:///c:/a/b/c/",
			cloudpath.WindowsFileSystem, "localhost", "c", "c:/a/b/c/", '/', nil,
		},
	}
	if err := testMatcherSpec(cloudpath.DefaultMatchers, data); err != nil {
		t.Errorf("%v", err)
	}
}

func TestEmpty(t *testing.T) {
	ms := cloudpath.MatcherSpec([]cloudpath.Matcher{})
	if ms.Match("a/b") != nil {
		t.Errorf("unexpected  match")
	}
	if ms.Scheme("a/b") != "" {
		t.Errorf("unexpected scheme")
	}
	if ms.Host("https://storage.cloud.google.com/a/b") != "" {
		t.Errorf("unexpected host")
	}
	if ms.Volume("s3://bucket/a/b") != "" {
		t.Errorf("unexpected volume")
	}
	if p, s := ms.Path("s3://bucket/a/b"); p != "" || s != utf8.RuneError {
		t.Errorf("unexpected path or separator")
	}
	if len(ms.Parameters("s3://bucket/a/b?p=a")) != 0 {
		t.Errorf("unexpected parameters")
	}
	if ms.IsLocal("/a/b") {
		t.Errorf("unexpected is local - should return false if matcher list is empty")
	}
}
