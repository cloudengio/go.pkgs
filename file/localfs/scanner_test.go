// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/internal"
	"cloudeng.io/file/localfs"
)

var localTestTree string

func TestMain(m *testing.M) {
	localTestTree = internal.CreateTestTree()
	code := m.Run()
	os.RemoveAll(localTestTree)
	os.Exit(code)
}

func scan(sc filewalk.FS, dir string) (dirNames, fileNames []string, errors []error, info map[string]file.Info) {
	ctx := context.Background()
	info = map[string]file.Info{}
	ds := sc.LevelScanner(dir)
	for ds.Scan(ctx, 1) {
		entries := ds.Contents()
		for _, entry := range entries {
			fi, err := sc.Lstat(ctx, sc.Join(dir, entry.Name))
			if err != nil {
				errors = append(errors, err)
				continue
			}
			info[entry.Name] = fi
			if entry.IsDir() {
				dirNames = append(dirNames, entry.Name)
			} else {
				fileNames = append(fileNames, entry.Name)
			}
		}
	}
	if err := ds.Err(); err != nil {
		errors = append(errors, err)
	}
	sort.Strings(dirNames)
	sort.Strings(fileNames)
	return
}

func TestLocalFilesystem(t *testing.T) {
	sc := localfs.New()

	dirs, files, errors, info := scan(sc, localTestTree)

	expectedDirNames := []string{"a0", "b0", "inaccessible-dir"}
	expectedFileNames := []string{"f0", "f1", "f2", "la0", "la1", "lf0"}

	if got, want := dirs, expectedDirNames; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := files, expectedFileNames; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := len(errors), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, d := range expectedDirNames {
		i := info[d]
		if got, want := i.IsDir(), true; got != want {
			t.Errorf("%v: got %v, want %v", d, got, want)
		}
		if got, want := i.Mode()&fs.ModeSymlink == fs.ModeSymlink, false; got != want {
			t.Errorf("%v: got %v, want %v", d, got, want)
		}
	}

	for _, f := range expectedFileNames {
		i := info[f]
		if got, want := i.IsDir(), false; got != want {
			t.Errorf("%v: got %v, want %v", f, got, want)
		}
		if !strings.HasPrefix(f, "l") {
			if got, want := i.Size(), int64(3); got != want {
				t.Errorf("%v: got %v, want %v", f, got, want)
			}
		}
		if got, want := i.Mode()&fs.ModeSymlink == fs.ModeSymlink, strings.HasPrefix(f, "l"); got != want {
			t.Errorf("%v: got %v, want %v", f, got, want)
		}
	}

	_, _, errors, _ = scan(sc, sc.Join(localTestTree, "inaccessible-dir"))

	if got, want := len(errors), 1; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got, want := os.IsPermission(errors[0]), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}

func TestScanCancelClosesFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	ds := localfs.New().LevelScanner(dir)
	if !ds.Scan(ctx, 1) {
		t.Fatalf("first Scan failed: %v", ds.Err())
	}
	cancel()
	if ds.Scan(ctx, 1) {
		t.Fatal("expected Scan to return false after context cancel")
	}
	if !errors.Is(ds.Err(), context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", ds.Err())
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("RemoveAll(%q): %v", dir, err)
	}
}

func manyFiles(t testing.TB, n int) string {
	t.Helper()
	dir := t.TempDir()
	for i := range n {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%06d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func scanNames(ctx context.Context, ds filewalk.LevelScanner, n int) ([]string, error) {
	var names []string
	for ds.Scan(ctx, n) {
		for _, e := range ds.Contents() {
			names = append(names, e.Name)
		}
	}
	return names, ds.Err()
}

func TestScanPrefetch(t *testing.T) {
	ctx := context.Background()
	dir := manyFiles(t, 105)
	want, err := scanNames(ctx, localfs.New(localfs.WithPrefetch(0)).LevelScanner(dir), 10)
	if err != nil || len(want) != 105 {
		t.Fatalf("got %v, %v", len(want), err)
	}
	sort.Strings(want)
	// Thresholds below, at and above the batch size, the directory size,
	// and a batch size that does not divide the directory.
	for _, min := range []int{1, 10, 20, 95, 105, 1000} {
		for _, n := range []int{1, 10, 50, 200} {
			ds := localfs.New(localfs.WithPrefetch(min)).LevelScanner(dir)
			got, err := scanNames(ctx, ds, n)
			if err != nil {
				t.Fatalf("min %v n %v: %v", min, n, err)
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("min %v n %v: got %v entries, want %v", min, n, len(got), len(want))
			}
		}
	}
}

func TestScanPrefetchCancel(t *testing.T) {
	dir := manyFiles(t, 100)
	ctx, cancel := context.WithCancel(context.Background())
	ds := localfs.New(localfs.WithPrefetch(5)).LevelScanner(dir)
	for range 2 {
		if !ds.Scan(ctx, 5) {
			t.Fatalf("scan failed: %v", ds.Err())
		}
	}
	cancel()
	if ds.Scan(ctx, 5) {
		t.Fatal("expected Scan to return false after context cancel")
	}
	if !errors.Is(ds.Err(), context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", ds.Err())
	}
}

func TestScanPrefetchClose(t *testing.T) {
	dir := manyFiles(t, 100)
	ds := localfs.New(localfs.WithPrefetch(5)).LevelScanner(dir)
	for range 2 {
		if !ds.Scan(context.Background(), 5) {
			t.Fatalf("scan failed: %v", ds.Err())
		}
	}
	if err := ds.(io.Closer).Close(); err != nil {
		t.Fatal(err)
	}
	if ds.Scan(context.Background(), 5) {
		t.Fatal("expected Scan to return false after Close")
	}
}

// BenchmarkScan reads a large directory, spending work on each batch as a
// walker handler would.
func BenchmarkScan(b *testing.B) {
	dir := manyFiles(b, 100000)
	ctx := context.Background()
	for _, work := range []time.Duration{0, 100 * time.Microsecond, 500 * time.Microsecond} {
		for _, min := range []int{0, localfs.DefaultPrefetch} {
			b.Run(fmt.Sprintf("work=%v/prefetch=%v", work, min), func(b *testing.B) {
				for b.Loop() {
					ds := localfs.New(localfs.WithPrefetch(min)).LevelScanner(dir)
					for ds.Scan(ctx, 1000) {
						_ = ds.Contents()
						if work > 0 {
							spin(work)
						}
					}
					if err := ds.Err(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func spin(d time.Duration) {
	for end := time.Now().Add(d); time.Now().Before(end); {
		runtime.Gosched()
	}
}
