// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs_test

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"cloudeng.io/aws/awstestutil"
	"cloudeng.io/aws/s3fs"
	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

var awsInstance *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsInstance,
		awstestutil.WithS3Tree("testdata/s3"))
}

func TestJoin(t *testing.T) {
	j := func(a ...string) []string {
		return a
	}
	for _, delim := range []byte{'/', '@'} {
		fs := s3fs.New(awstestutil.DefaultAWSConfig(), s3fs.WithDelimiter(delim))
		for i, tc := range []struct {
			input  []string
			output string
		}{
			{},
			{j("a", "b"), "a/b"},
			{j("a", "b", "c"), "a/b/c"},
			{j("s3://a", "b"), "s3://a/b"},
			{j("s3://a", "", "b"), "s3://a/b"},
			{j("s3://a", "b/", "c/"), "s3://a/b/c/"},
			{j("s3://a", "/b", "/c"), "s3://a/b/c"},
			{j("s3://a/", "b/", "c/"), "s3://a/b/c/"},
		} {
			in := []string{}
			for _, i := range tc.input {
				in = append(in, strings.ReplaceAll(i, "/", string(delim)))
			}
			out := strings.ReplaceAll(tc.output, "/", string(delim))
			if got, want := fs.Join(in...), out; got != want {
				t.Errorf("%v: got %v, want %v", i, got, want)
			}
		}
	}
}

type s3walker struct {
	fs       filewalk.FS
	prefixes []string
	contents []string
}

func (w *s3walker) Prefix(_ context.Context, state *struct{}, prefix string, _ file.Info, err error) (bool, file.InfoList, error) {
	w.prefixes = append(w.prefixes, prefix)
	return false, nil, nil
}

func (w *s3walker) Contents(ctx context.Context, state *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		key := w.fs.Join(prefix, c.Name)
		if !c.IsDir() {
			w.contents = append(w.contents, key)
			continue
		}
		info, err := w.fs.Stat(ctx, key)
		if err != nil {
			return nil, err
		}
		children = append(children, info)
	}
	return children, nil
}

func (w *s3walker) Done(_ context.Context, state *struct{}, prefix string, err error) error {
	return err
}

func walkAndCompare(ctx context.Context, t *testing.T, fs filewalk.FS, start, prefixes, contents []string) {
	t.Helper()
	w := &s3walker{fs: fs}
	walker := filewalk.New(fs, w)
	if err := walker.Walk(ctx, start...); err != nil {
		t.Fatal(err)
	}
	sort.Strings(w.prefixes)
	if got, want := w.prefixes, prefixes; !slices.Equal(got, want) {
		t.Errorf("prefixes: got %v, want %v", got, want)
	}
	sort.Strings(w.contents)
	if got, want := w.contents, contents; !slices.Equal(got, want) {
		t.Errorf("contents: got %v, want %v", got, want)
	}
}

func newS3FS() filewalk.FS {
	cfg := awstestutil.DefaultAWSConfig()
	return s3fs.New(cfg, s3fs.WithS3Client(awsInstance.S3(cfg)))
}

func newS3ObjFS() *s3fs.T {
	cfg := awstestutil.DefaultAWSConfig()
	return s3fs.NewS3FS(cfg,
		s3fs.WithS3Client(awsInstance.S3(cfg)),
		s3fs.WithScanSize(2),
	)
}

func scanAndCompare(ctx context.Context, t *testing.T, fs filewalk.FS, start string, contents []string) {
	t.Helper()
	sc := fs.LevelScanner(start)
	found := []string{}
	for sc.Scan(ctx, 1) {
		for _, c := range sc.Contents() {
			found = append(found, fs.Join(start, c.Name))
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if got, want := found, contents; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestScan(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3FS()
	scanAndCompare(ctx, t, fs, "s3://bucket-a", []string{
		"s3://bucket-a/0",
		"s3://bucket-a/1",
		"s3://bucket-a/2",
		"s3://bucket-a/a/",
		"s3://bucket-a/b/",
		"s3://bucket-a/c/",
	})
	scanAndCompare(ctx, t, fs, "s3://bucket-b", []string{
		"s3://bucket-b/0",
		"s3://bucket-b/1",
		"s3://bucket-b/2",
		"s3://bucket-b/x/",
		"s3://bucket-b/y/",
	})
	scanAndCompare(ctx, t, fs, "s3://bucket-b/x/", []string{
		"s3://bucket-b/x/y/",
	})
	scanAndCompare(ctx, t, fs, "s3://bucket-b/x/y/", []string{
		"s3://bucket-b/x/y/0",
		"s3://bucket-b/x/y/z/",
	})
	scanAndCompare(ctx, t, fs, "s3://bucket-b/x/y/z/", []string{
		"s3://bucket-b/x/y/z/0",
	})
}

func TestStat(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3FS()

	info, err := fs.Stat(ctx, "s3://bucket-a/0")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Name(), "0"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestWalk(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3FS()

	walkAndCompare(ctx, t, fs,
		[]string{"s3://bucket-a",
			"s3://bucket-b",
			"s3://bucket-c"},
		[]string{
			"s3://bucket-a",
			"s3://bucket-a/a/",
			"s3://bucket-a/b/",
			"s3://bucket-a/c/",
			"s3://bucket-b",
			"s3://bucket-b/x/",
			"s3://bucket-b/x/y/",
			"s3://bucket-b/x/y/z/",
			"s3://bucket-b/y/",
			"s3://bucket-c",
		},
		[]string{"s3://bucket-a/0",
			"s3://bucket-a/1",
			"s3://bucket-a/2",
			"s3://bucket-a/a/0",
			"s3://bucket-a/a/1",
			"s3://bucket-a/a/2",
			"s3://bucket-a/b/0",
			"s3://bucket-a/b/1",
			"s3://bucket-a/b/2",
			"s3://bucket-a/c/0",
			"s3://bucket-a/c/1",
			"s3://bucket-a/c/2",
			"s3://bucket-b/0",
			"s3://bucket-b/1",
			"s3://bucket-b/2",
			"s3://bucket-b/x/y/0",
			"s3://bucket-b/x/y/z/0",
			"s3://bucket-b/y/0",
			"s3://bucket-c/0",
			"s3://bucket-c/1",
			"s3://bucket-c/2",
		},
	)
}

func TestPutGet(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3ObjFS()

	obj, err := fs.Get(ctx, "s3://bucket-a/a/2")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(obj), "2\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if err := fs.Put(ctx, "s3://bucket-a/a/b/c/32", 0x00, []byte("32\n")); err != nil {
		t.Fatal(err)
	}

	obj, err = fs.Get(ctx, "s3://bucket-a/a/b/c/32")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(obj), "32\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDelete(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3ObjFS()
	prefix := "s3://bucket-b/a"
	for i, name := range []string{
		"c/d/1",
		"e/e/32",
		"e/e/33",
		"c/d/2",
		"c/d/3",
		"g/g/4",
	} {
		p := fs.Join(prefix, name)
		if err := fs.Put(ctx, p, 0x00, []byte(fmt.Sprintf("%03v\n", i))); err != nil {
			t.Fatal(err)
		}
	}

	scanAndCompare(ctx, t, fs, prefix+"/", []string{
		"s3://bucket-b/a/c/",
		"s3://bucket-b/a/e/",
		"s3://bucket-b/a/g/",
	})

	walkAndCompare(ctx, t, fs, []string{prefix + "/"}, []string{
		"s3://bucket-b/a/",
		"s3://bucket-b/a/c/",
		"s3://bucket-b/a/c/d/",
		"s3://bucket-b/a/e/",
		"s3://bucket-b/a/e/e/",
		"s3://bucket-b/a/g/",
		"s3://bucket-b/a/g/g/",
	}, []string{
		"s3://bucket-b/a/c/d/1",
		"s3://bucket-b/a/c/d/2",
		"s3://bucket-b/a/c/d/3",
		"s3://bucket-b/a/e/e/32",
		"s3://bucket-b/a/e/e/33",
		"s3://bucket-b/a/g/g/4",
	})

	if err := fs.Delete(ctx, fs.Join(prefix, "/e/e/32")); err != nil {
		t.Fatal(err)
	}

	walkAndCompare(ctx, t, fs, []string{prefix + "/"}, []string{
		"s3://bucket-b/a/",
		"s3://bucket-b/a/c/",
		"s3://bucket-b/a/c/d/",
		"s3://bucket-b/a/e/",
		"s3://bucket-b/a/e/e/",
		"s3://bucket-b/a/g/",
		"s3://bucket-b/a/g/g/",
	}, []string{
		"s3://bucket-b/a/c/d/1",
		"s3://bucket-b/a/c/d/2",
		"s3://bucket-b/a/c/d/3",
		"s3://bucket-b/a/e/e/33",
		"s3://bucket-b/a/g/g/4",
	})

	if err := fs.DeleteAll(ctx, fs.Join(prefix, "/c")); err != nil {
		t.Fatal(err)
	}

	walkAndCompare(ctx, t, fs, []string{prefix + "/"}, []string{
		"s3://bucket-b/a/",
		"s3://bucket-b/a/e/",
		"s3://bucket-b/a/e/e/",
		"s3://bucket-b/a/g/",
		"s3://bucket-b/a/g/g/",
	}, []string{
		"s3://bucket-b/a/e/e/33",
		"s3://bucket-b/a/g/g/4",
	})
}

func TestErrors(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3ObjFS()
	// S3 silently ignores deletions of non-existing objects.
	if err := fs.Delete(ctx, fs.Join("s3://bucket-a", "nothere")); err != nil {
		t.Fatal(err)
	}
	_, err := fs.Get(ctx, "s3://bucket-a/nothere")
	if !fs.IsNotExist(err) {
		t.Errorf("unexpected or missing error: %v", err)
	}

	_, err = fs.OpenCtx(ctx, "s3://bucket-a/nothere")
	if !fs.IsNotExist(err) {
		t.Errorf("unexpected or missing error: %v", err)
	}
}
