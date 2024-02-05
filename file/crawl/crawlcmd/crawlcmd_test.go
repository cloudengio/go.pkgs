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

	"cloudeng.io/file"
	"cloudeng.io/file/content"
	"cloudeng.io/file/crawl/crawlcmd"
	"cloudeng.io/file/crawl/outlinks"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/file/filewalk/filewalktestutil"
	"cloudeng.io/file/filewalk/localfs"
	"cloudeng.io/path"
)

type randfs struct{}

func (f *randfs) NewFS(ctx context.Context) (file.FS, error) {
	src := rand.NewSource(time.Now().UnixMicro())
	return filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 1024)), nil
}

func expectedOutput(fs file.FS, name, root string, seeds ...string) (dirs, files []string) {
	dirs = []string{root, fs.Join(root, "crawled"), fs.Join(root, "checkpoint")}
	sharder := path.NewSharder(path.WithSHA1PrefixLength(1))
	for _, seed := range seeds {
		p, f := sharder.Assign(name + seed)
		dirs = append(dirs, fs.Join(root, "crawled", p))
		files = append(files, fs.Join(root, "crawled", p, f))
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
		Extractors: func() map[content.Type]outlinks.Extractor {
			return map[content.Type]outlinks.Extractor{}
		},
	}

	writeFS := file.LocalFS()
	writeRoot := tmpDir

	fsMap := map[string]file.FSFactory{
		"unix": &randfs{},
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
	cachePath, checkpointPath, err := cmd.Cache.InitStore(ctx, writeFS, writeRoot)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := cachePath, filepath.Join(writeRoot, "crawled"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := checkpointPath, filepath.Join(writeRoot, "checkpoint"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	err = cmd.Run(ctx, fsMap, writeRoot, writeFS, false, false)
	if err != nil {
		t.Fatal(err)
	}

	expectedDirs, expectedFiles := expectedOutput(writeFS, cmd.Config.Name, writeRoot, cmd.Config.Seeds...)

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
	cachePath, checkpointPath, err = cmd.Cache.InitStore(ctx, writeFS, writeRoot)
	if err != nil {
		t.Fatal(err)
	}
	prefixes, contents, err = filewalktestutil.WalkContents(ctx, lfs, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(prefixes), 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := len(contents), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
