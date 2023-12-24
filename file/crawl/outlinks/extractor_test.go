// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks_test

import (
	"bytes"
	"context"
	"embed"
	"io"
	"io/fs"
	"path"
	"reflect"
	"sync"
	"testing"

	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/download"
	"cloudeng.io/file/filetestutil"
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
			FS: filetestutil.WrapEmbedFS(htmlExamples),
		},
		Downloads: []download.Result{
			{Name: path.Join("testdata", name), Contents: buf.Bytes()},
		},
	}
}

func collectErrors(ch <-chan outlinks.Errors) []outlinks.Errors {
	errs := []outlinks.Errors{}
	for err := range ch {
		errs = append(errs, err)
	}
	return errs
}

func TestGenericExtractor(t *testing.T) {
	ctx := context.Background()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	errs := []outlinks.Errors{}
	errCh := make(chan outlinks.Errors, 10)
	go func() {
		errs = collectErrors(errCh)
		wg.Done()
	}()

	downloaded := downloadFromTestdata(t, "simple.html")

	reg := content.NewRegistry[outlinks.Extractor]()
	if err := reg.RegisterHandlers("text/html;charset=utf-8", outlinks.NewHTML()); err != nil {
		t.Fatal(err)
	}

	ext := outlinks.NewExtractors(errCh, &outlinks.PassthroughProcessor{}, reg)
	reqs := ext.Extract(ctx, 0, downloaded)
	reqs2 := ext.Extract(ctx, 0, downloaded)
	close(errCh)
	wg.Wait()

	for _, err := range errs {
		for _, detail := range err.Errors {
			if detail.Error != nil {
				t.Fatal(err)
			}
		}
	}

	extracted := []string{}
	for _, req := range reqs {
		extracted = append(extracted, req.Names()...)
	}
	if got, want := extracted, []string{
		"https://www.w3.org/",
		"https://www.google.com/",
		"/testdata/html_images.asp",
		"/css/default.asp",
		"https://sample.css",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// duplicates should be ignored.
	if got, want := len(reqs2), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
