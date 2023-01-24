// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd_test

import (
	"reflect"
	"testing"

	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl/crawlcmd"
	"gopkg.in/yaml.v3"
)

const crawlsSpec = `
  name: test
  depth: 3
  seeds:
    - s3://foo/bar
    - https://yahoo.com

  download:
    default_concurrency: 4 # 0 will default to all available CPUs
    default_request_chan_size: 100
    default_downloads_chan_size: 100
    per_depth_concurrency: [1, 2, 4]
  num_extractors: 3
  extractors:
    text/html;charset=utf-8: html
    text/plain;charset=utf-8: text
  `

func TestCrawlConfig(t *testing.T) {
	var crawl crawlcmd.Config
	if err := yaml.Unmarshal([]byte(crawlsSpec), &crawl); err != nil {
		t.Fatal(err)
	}

	if got, want := crawl.Name, "test"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(crawl.Seeds), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := crawl.Depth, 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := crawl.Download.DefaultConcurrency, 4; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := crawl.Download.PerDepthConcurrency, []int{1, 2, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := crawl.NumExtractors, 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := crawl.Extractors, map[content.Type]string{"text/html;charset=utf-8": "html", "text/plain;charset=utf-8": "text"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
