// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"cloudeng.io/cmdutil"
	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl/crawlcmd"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/path/cloudpath"
)

const crawlsSpec = `
  name: test
  depth: 3
  seeds:
    - s3://foo/bar
    - https://yahoo.com
    - s3://baz

  download:
    default_concurrency: 4 # 0 will default to all available CPUs
    default_request_chan_size: 100
    default_downloads_chan_size: 100
    per_depth_concurrency: [1, 2, 4]
  num_extractors: 3
  extractors: [text/html;charset=utf-8, text/plain;charset=utf-8]
  `

func TestCrawlConfig(t *testing.T) {
	var crawl crawlcmd.Config
	if err := cmdutil.ParseYAMLConfigString(crawlsSpec, &crawl); err != nil {
		t.Fatal(err)
	}

	if got, want := crawl.Name, "test"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(crawl.Seeds), 3; got != want {
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

	if got, want := crawl.Extractors, []content.Type{"text/html;charset=utf-8", "text/plain;charset=utf-8"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

type dummyFSFactory struct {
	called string
}

func (d *dummyFSFactory) New(ctx context.Context, scheme string) (file.FS, error) {
	d.called = scheme
	return filetestutil.NewMockFS(
		filetestutil.FSScheme(scheme),
		filetestutil.FSWithConstantContents([]byte{'a'}, 10)), nil
}

func (d *dummyFSFactory) NewFromMatch(ctx context.Context, match cloudpath.Match) (file.FS, error) {
	return d.New(ctx, match.Scheme)
}

func TestCrawlSeeds(t *testing.T) {
	ctx := context.Background()
	var crawl crawlcmd.Config
	crawl.Seeds = []string{"https://yahoo.com", "s3://foo/bar", "s3://baz", "c:/foo/bar"}

	byScheme, rejected := crawl.SeedsByScheme(cloudpath.DefaultMatchers)
	if got, want := len(byScheme), 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(rejected), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	s3Only := []cloudpath.Matcher{cloudpath.AWSS3Matcher}
	byScheme, rejected = crawl.SeedsByScheme(s3Only)
	if got, want := len(byScheme), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(byScheme["s3"]), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := rejected, []string{"https://yahoo.com", "c:/foo/bar"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	s3HTTP := []cloudpath.Matcher{cloudpath.AWSS3Matcher, cloudpath.URLMatcher}
	byScheme, rejected = crawl.SeedsByScheme(s3HTTP)
	if got, want := len(byScheme), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	factories := map[string]file.FSFactory{
		"s3":    &dummyFSFactory{},
		"https": &dummyFSFactory{},
	}

	requests, err := crawl.CreateSeedCrawlRequests(ctx, factories, byScheme)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(requests), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, req := range requests {
		fmt.Printf("container... %v\n", req.Container().Scheme())
		if req.Container().Scheme() == "s3" {
			if got, want := req.Names(), []string{"s3://foo/bar", "s3://baz"}; !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
		if req.Container().Scheme() == "https" {
			if got, want := req.Names(), []string{"https://yahoo.com"}; !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}

	if got, want := factories["s3"].(*dummyFSFactory).called, "s3"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := factories["https"].(*dummyFSFactory).called, "https"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
