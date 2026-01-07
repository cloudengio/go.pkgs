// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/localfs"
)

var _ file.FS = (*localfs.R)(nil)
var _ file.FS = (*localfs.T)(nil)
var _ filewalk.FS = (*localfs.R)(nil)
var _ filewalk.FS = (*localfs.T)(nil)

func TestXAttr(t *testing.T) {
	tmpdir := t.TempDir()

	ctx := context.Background()
	fs := localfs.New()
	name := fs.Join(tmpdir, "testfile")
	// #nosec G306
	if err := os.WriteFile(name, make([]byte, 4096), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := fs.Stat(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	xattr, err := fs.XAttr(ctx, name, info)
	if err != nil {
		t.Fatal(err)
	}
	if xattr.Device == 0 || xattr.FileID == 0 {
		t.Fatalf("got %v, want non-zero", xattr)
	}

	if got, want := xattr.Blocks, int64(8); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	uid, gid := os.Getuid(), os.Getgid()
	if got, want := xattr.UID, int64(uid); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := xattr.GID, int64(gid); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := xattr.Hardlinks, uint64(1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if xattr.UID == -1 && len(xattr.User) == 0 {
		t.Errorf("got %v, want non-empty User", xattr)
	}

	if xattr.UID == -1 && len(xattr.Group) == 0 {
		t.Errorf("got %v, want non-empty Group", xattr)
	}

}

func TestSetXAttr(t *testing.T) {
	x := file.XAttr{
		UID:       1,
		GID:       2,
		User:      "user",
		Group:     "group",
		Device:    3,
		FileID:    4,
		Blocks:    5,
		Hardlinks: 6,
	}
	now := time.Now()
	fi := file.NewInfo("test", 8, 0, now, x)

	if got, want := fi.Sys(), x; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestLargeFile(t *testing.T) {
	ctx := context.Background()
	tmpdir := t.TempDir()
	name := filepath.Join(tmpdir, "largefile")
	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	if err := os.WriteFile(name, content, 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	blockSize := 10
	digest, _ := digests.New("sha-256", []byte("test-digest"))
	lf, err := localfs.NewLargeFile(f, blockSize, digest)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := lf.Name(), name; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	size, bs := lf.ContentLengthAndBlockSize()
	if got, want := size, int64(len(content)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := bs, blockSize; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := lf.Digest(), digest; got.Algo != want.Algo {
		t.Errorf("got %v, want %v", got, want)
	}

	// Test GetReader
	from, to := int64(5), int64(15)
	rd, retry, err := lf.GetReader(ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if retry.IsRetryable() {
		t.Errorf("expected not to be retryable")
	}

	// The returned reader is not limited by 'to', so use a LimitReader.
	readContent, err := io.ReadAll(io.LimitReader(rd, to-from+1))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(readContent), string(content[from:to+1]); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if err := rd.Close(); err != nil {
		t.Errorf("close failed: %v", err)
	}
}

func TestLocalFS(t *testing.T) {
	ctx := context.Background()
	tmpdir := t.TempDir()
	fs := localfs.New()

	// Test Join and Base
	path := fs.Join("a", "b", "c")
	if got, want := path, filepath.Join("a", "b", "c"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fs.Base(path), "c"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// WriteFile
	name := filepath.Join(tmpdir, "testfile")
	content := []byte("hello world")
	if err := fs.WriteFile(name, content, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// ReadFile
	readContent, err := fs.ReadFile(name)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if got, want := string(readContent), string(content); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Stat
	info, err := fs.Stat(ctx, name)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if got, want := info.Name(), "testfile"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := info.Size(), int64(len(content)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Delete
	if err := fs.Delete(ctx, name); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := fs.Stat(ctx, name); !fs.IsNotExist(err) {
		t.Errorf("expected NotExist error, got %v", err)
	}
}

func TestRootedFS(t *testing.T) {
	ctx := context.Background()
	tmpdir := t.TempDir()
	root := filepath.Join(tmpdir, "root")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}

	rfs := localfs.NewTree(root)

	// Test Join -> should join relative paths
	if got, want := rfs.Join("a", "b"), filepath.Join("a", "b"); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// WriteFile using relative path
	name := "testfile"
	content := []byte("rooted data")
	if err := rfs.WriteFile(name, content, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file exists on disk at expected location
	diskPath := filepath.Join(root, name)
	if _, err := os.Stat(diskPath); err != nil {
		t.Errorf("file not found on disk at %v: %v", diskPath, err)
	}

	// ReadFile
	readContent, err := rfs.ReadFile(name)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if got, want := string(readContent), string(content); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Stat
	info, err := rfs.Stat(ctx, name)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if got, want := info.Name(), "testfile"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// EnsurePrefix (MkdirAll)
	subDir := "sub/dir"
	if err := rfs.EnsurePrefix(ctx, subDir, 0700); err != nil {
		t.Fatalf("EnsurePrefix failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, subDir)); err != nil {
		t.Errorf("directory not created on disk: %v", err)
	}

	// Delete
	if err := rfs.Delete(ctx, name); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
		t.Errorf("file still exists on disk")
	}
}
