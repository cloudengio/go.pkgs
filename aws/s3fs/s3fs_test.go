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
	for _, delim := range []string{"/", "@"} {
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
	fmt.Printf("prefix.....: %v: %v\n", prefix, err)
	w.prefixes = append(w.prefixes, prefix)
	return false, nil, nil
}

func (w *s3walker) Contents(ctx context.Context, state *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	fmt.Printf("contents: %v %v\n", prefix, len(contents))
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		key := w.fs.Join(prefix, c.Name)
		if !c.IsDir() {
			fmt.Printf("Obj: %v %v %v\n", prefix, c.Name, key)
			w.contents = append(w.contents, key)
			continue
		}
		info, err := w.fs.Stat(ctx, key)
		if err != nil {
			return nil, err
		}
		children = append(children, info)
		fmt.Printf("CHILD: %v\n", info)
	}
	return children, nil
}

func (w *s3walker) Done(_ context.Context, state *struct{}, prefix string, err error) error {
	return err
}

func newS3FS() filewalk.FS {
	cfg := awstestutil.DefaultAWSConfig()
	return s3fs.New(cfg, s3fs.WithS3Client(awsInstance.S3(cfg)))
}

func TestScan(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	fs := newS3FS()
	parent := "s3://bucket-a"
	sc := fs.LevelScanner(parent)
	found := []string{}
	for sc.Scan(ctx, 1) {
		for _, c := range sc.Contents() {
			fmt.Printf("PARENT: %v %v\n", parent, c.Name)
			found = append(found, fs.Join(parent, c.Name))
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if got, want := found, []string{
		"s3://bucket-a/0",
		"s3://bucket-a/1",
		"s3://bucket-a/2",
		"s3://bucket-a/a/",
		"s3://bucket-a/b/",
		"s3://bucket-a/c/",
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
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

	w := &s3walker{fs: fs}
	walker := filewalk.New(fs, w)
	if err := walker.Walk(ctx, "s3://bucket-a"); err != nil {
		t.Fatal(err)
	}
	sort.Strings(w.prefixes)
	if got, want := w.prefixes, []string{
		"s3://bucket-a",
		"s3://bucket-a/a/",
		"s3://bucket-a/b/",
		"s3://bucket-a/c/",
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	sort.Strings(w.contents)
	if got, want := w.contents, []string{
		"s3://bucket-a/0",
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
	}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	fmt.Printf("%#v\n", w)

}

/*
//go:embed testdata
var testdata embed.FS

func TestS3FS(t *testing.T) {
	ctx := context.Background()

	mfs := s3fstestutil.NewMockFS(filetestutil.WrapEmbedFS(testdata),
		s3fstestutil.WithBucket("bucket"),
		s3fstestutil.WithLeadingSlashStripped())
	fs := s3fs.New(aws.Config{}, s3fs.WithS3Client(mfs))

	name := "example.html"
	fi, err := fs.OpenCtx(ctx, "s3://"+path.Join("bucket", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(fi)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}
}
*/
