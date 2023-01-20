// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks_test

import (
	"bytes"
	"embed"
	"io"
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

func loadTestdata(t *testing.T, name string) fs.File {
	f, err := htmlExamples.Open(path.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func downloadFromTestdata(t *testing.T, name string) download.Downloaded {
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, loadTestdata(t, name)); err != nil {
		t.Fatal(err)
	}
	return download.Downloaded{
		Request: download.SimpleRequest{
			FS: file.WrapFS(htmlExamples),
		},
		Downloads: []download.Result{
			{Name: path.Join("testdata", name), Contents: buf.Bytes()},
		},
	}
}

func TestHTML(t *testing.T) {
	var he outlinks.HTML
	rd := loadTestdata(t, "simple.html")
	defer rd.Close()
	extracted, err := he.HREFs("", rd)
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
}
