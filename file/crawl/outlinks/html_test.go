// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks_test

import (
	"embed"
	"io/fs"
	"path"
	"reflect"
	"testing"

	"cloudeng.io/file"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
)

//go:embed testdata/*.html
var htmlExamples embed.FS

func loadTestdata(name string) fs.File {
	f, err := htmlExamples.Open(path.Join("testdata", name))
	if err != nil {
		panic(err)
	}
	return f
}

func downloadFromTestdata(name string) download.Downloaded {
	return download.Downloaded{
		Container: file.FSFromFS(htmlExamples),
		Downloads: []download.Result{
			{Name: path.Join("testdata", name)},
		},
	}
}

func TestHTML(t *testing.T) {
	var he outlinks.HTML
	rd := loadTestdata("simple.html")
	defer rd.Close()
	extracted, err := he.HREFs(rd)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := extracted, []string{
		"https://www.w3.org/",
		"https://www.google.com/",
		"html_images.asp",
		"/css/default.asp",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
