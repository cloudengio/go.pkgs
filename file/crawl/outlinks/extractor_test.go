// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks_test

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"cloudeng.io/file/crawl/outlinks"
)

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

	downloaded := downloadFromTestdata("simple.html")

	ext := outlinks.NewExtractor(outlinks.NewHTML(), errCh)
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
		"html_images.asp",
		"/css/default.asp",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// duplicates should be ignored.
	if got, want := len(reqs2), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
