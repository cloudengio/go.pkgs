// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package extractors_test

import (
	"context"
	"embed"
	"reflect"
	"testing"

	"cloudeng.io/file/crawl/extractors"
	"cloudeng.io/file/download"
)

//go:embed testdata/*.html
var htmlExamples embed.FS

func downloadsFromTestdata() download.Downloaded {
	return download.Downloaded{
		Container: htmlExamples,
		Downloads: []download.Result{
			{
				Name: "testdata/simple.html",
			},
		},
	}
}

func TestHTML(t *testing.T) {
	ctx := context.Background()
	errCh := make(chan extractors.Errors, 10)
	he := extractors.NewHTML(errCh)

	errs := []extractors.Errors{}

	go func() {
		for err := range errCh {
			errs = append(errs, err)
		}
	}()
	extracted := he.Extract(ctx, 1, downloadsFromTestdata())
	if got, want := extracted, []string{
		"https://www.w3.org/",
		"https://www.google.com/",
		"html_images.asp",
		"/css/default.asp",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
