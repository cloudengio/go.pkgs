// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdyaml"
	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl/crawlcmd"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/file/filewalk/filewalktestutil"
	"cloudeng.io/file/localfs"
	"cloudeng.io/path"
	"gopkg.in/yaml.v3"
)

type randfs struct{}

func (f *randfs) NewFS(_ context.Context) (file.FS, error) {
	src := rand.NewSource(time.Now().UnixMicro())
	return filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 1024)), nil
}

func expectedOutput(fs file.FS, name, root, downloads string, seeds ...string) (dirs, files []string) {
	dirs = []string{root, downloads}
	sharder := path.NewSharder(path.WithSHA1PrefixLength(1))
	for _, seed := range seeds {
		p, f := sharder.Assign(name + seed)
		dirs = append(dirs, fs.Join(downloads, p))
		files = append(files, fs.Join(downloads, p, f))
	}
	sort.Strings(dirs)
	sort.Strings(files)
	return
}

func TestCrawlCmd(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "crawlcmd")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if t.Failed() {
			fmt.Printf("tmpDir: %s\n", tmpDir)
		} else {
			os.RemoveAll(tmpDir)
		}
	}()

	cmd := crawlcmd.Crawler{
		Extractors: map[content.Type]outlinks.Extractor{},
	}

	writeFS := localfs.New()
	writeRoot := tmpDir

	fsMap := map[string]crawlcmd.FSFactory{
		"unix": (&randfs{}).NewFS,
	}

	seeds := []string{"rand1", "rand6"}

	cmd.Config = crawlcmd.Config{
		Name:  "test",
		Depth: 0,
		Seeds: seeds,
		Download: crawlcmd.DownloadConfig{
			DownloadFactoryConfig: crawlcmd.DownloadFactoryConfig{
				DefaultConcurrency: 1,
			},
		},
		Cache: crawlcmd.CrawlCacheConfig[yaml.Node]{
			Downloads:         filepath.Join(writeRoot, "crawled"),
			Checkpoint:        filepath.Join(writeRoot, "checkpoint"),
			ClearBeforeCrawl:  true,
			ShardingPrefixLen: 1,
			Concurrency:       2,
		},
	}

	downloads, _ := cmd.Cache.Paths()
	if err := cmd.Cache.PrepareDownloads(ctx, writeFS); err != nil {
		t.Fatal(err)
	}

	err = cmd.Run(ctx, fsMap, writeFS, false, false)
	if err != nil {
		t.Fatal(err)
	}

	expectedDirs, expectedFiles := expectedOutput(writeFS, cmd.Config.Name,
		writeRoot, downloads, cmd.Config.Seeds...)

	lfs := localfs.New()
	prefixes, contents, err := filewalktestutil.WalkContents(ctx, lfs, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(prefixes)
	sort.Strings(contents)

	if got, want := prefixes, expectedDirs; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := contents, expectedFiles; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Test erase.
	cmd.Config.Cache.ClearBeforeCrawl = true
	if err := cmd.Cache.PrepareDownloads(ctx, writeFS); err != nil {
		t.Fatal(err)
	}
	prefixes, contents, err = filewalktestutil.WalkContents(ctx, lfs, tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(prefixes), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(contents), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func ExampleCrawlCacheConfig() {
	type cloudConfig struct {
		Region string `yaml:"region"`
	}
	var cfg crawlcmd.CrawlCacheConfig[yaml.Node]
	var service crawlcmd.CrawlCacheConfig[cloudConfig]

	err := cmdyaml.ParseConfig([]byte(`
cache_root: cloud-service://bucket/downloads
cache_service_config:
  region: us-west-2
`), &cfg)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	if err := crawlcmd.ParseCrawlCacheConfig(cfg, &service); err != nil {
		fmt.Printf("error: %v\n", err)
	}
	fmt.Println(service.Downloads)
	fmt.Println(service.ServiceConfig.Region)
	// Output:
	// cloud-service://bucket/downloads
	// us-west-2

}
