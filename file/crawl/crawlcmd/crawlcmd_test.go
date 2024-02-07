// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawlcmd_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"sort"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl/crawlcmd"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/file/filewalk/filewalktestutil"
	"cloudeng.io/file/localfs"
	"cloudeng.io/path"
)

type randfs struct{}

func (f *randfs) NewFS(_ context.Context) (file.FS, error) {
	src := rand.NewSource(time.Now().UnixMicro())
	return filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 1024)), nil
}

func expectedOutput(fs file.FS, name, root, cache string, seeds ...string) (dirs, files []string) {
	dirs = []string{root, fs.Join(root, cache)}
	sharder := path.NewSharder(path.WithSHA1PrefixLength(1))
	for _, seed := range seeds {
		p, f := sharder.Assign(name + seed)
		dirs = append(dirs, fs.Join(root, cache, p))
		files = append(files, fs.Join(root, cache, p, f))
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
		Cache: crawlcmd.CrawlCacheConfig{
			Prefix:            "crawled",
			Checkpoint:        "checkpoint",
			ClearBeforeCrawl:  true,
			ShardingPrefixLen: 1,
		},
	}
	cachePath, err := cmd.Cache.InitStore(ctx, writeFS, writeRoot)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := cachePath, cmd.Config.Cache.Prefix; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	err = cmd.Run(ctx, fsMap, writeRoot, writeFS, false, false)
	if err != nil {
		t.Fatal(err)
	}

	expectedDirs, expectedFiles := expectedOutput(writeFS, cmd.Config.Name,
		writeRoot, cachePath, cmd.Config.Seeds...)

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
	_, err = cmd.Cache.InitStore(ctx, writeFS, writeRoot)
	if err != nil {
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
