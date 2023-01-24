// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package processors_test

import (
	"embed"
	"io/fs"
	"path"
	"reflect"
	"testing"

	"cloudeng.io/file/content/processors"
)

//go:embed testdata/*.html
var htmlExamples embed.FS

func loadTestdata(t *testing.T, name string) fs.File {
	f, err := htmlExamples.Open(path.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestHTML(t *testing.T) {
	var he processors.HTML
	rd := loadTestdata(t, "simple.html")
	defer rd.Close()
	doc, err := he.Parse(rd)
	if err != nil {
		t.Fatal(err)
	}
	extracted, err := doc.HREFs("")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := extracted, []string{
		"https://www.w3.org/",
		"https://www.google.com/",
		"/html_images.asp",
		"/css/default.asp",
		"https://sample.css",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := doc.Title(), "My Title"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
