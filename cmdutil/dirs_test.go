package cmdutil_test

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/cmdutil"
	"cloudeng.io/errors"
)

func list(t *testing.T, root string) ([]string, []string, []string) {
	var names, perms, shas []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if path == root {
			return nil
		}
		if err != nil {
			return err
		}
		names = append(names, path)
		perms = append(perms, info.Mode().Perm().String())
		if info.IsDir() {
			s := sha1.Sum([]byte(strings.TrimPrefix(path, root)))
			shas = append(shas, hex.EncodeToString(s[:]))
			fmt.Printf("P: %v %v\n", strings.TrimPrefix(path, root), hex.EncodeToString(s[:]))
			return err
		}
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		s := sha1.Sum(buf)
		shas = append(shas, hex.EncodeToString(s[:]))
		return nil
	})
	if err != nil {
		t.Fatalf("filepath.Walk: %v: %v", root, err)
	}
	return names, perms, shas
}

func cmplists(t *testing.T, a, b []string, suffix bool) {
	if got, want := len(a), len(b); got != want {
		t.Errorf("%v: got %v, want %v", errors.Caller(2, 1), got, want)
		return
	}
	for i := range a {
		if suffix {
			if got, want := a[i], b[i]; !strings.HasSuffix(got, want) {
				t.Errorf("%v: got %v does not have suffix %v", errors.Caller(2, 1), got, want)
			}
			continue
		}
		if got, want := a[i], b[i]; got != want {
			t.Errorf("%v: got %v, want %v", errors.Caller(2, 1), got, want)
		}
	}
}

func TestMirrorDirTree(t *testing.T) {
	td, err := ioutil.TempDir("", "test-mirror")
	t.Logf("testdir: %v", td)
	defer os.RemoveAll(td)
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	if err := cmdutil.CopyAll("testdata", td, false); err != nil {
		t.Fatalf("MirrorDirTree: %v", err)
	}
	paths, perms, shas := list(t, td)
	expectedPaths := []string{
		"testdata",
		"testdata/a",
		"testdata/a/b",
		"testdata/a/b/c",
		"testdata/a/b/c/d",
		"testdata/a/b/c/f5",
		"testdata/a/b/c/f6",
		"testdata/a/b/d",
		"testdata/a/b/f3",
		"testdata/a/b/f4",
		"testdata/a/d",
		"testdata/a/f1",
		"testdata/a/f2",
	}
	expectedPerms := []string{
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rw-r--r--",
		"-rw-r--r--",
		"-rwxr-xr-x",
		"-rw-r--r--",
		"-rw-r--r--",
		"-rwxr-xr-x",
		"-rw-r--r--",
		"-rw-r--r--",
	}
	expectedShas := []string{
		"7956815567b5ab861b991832a34b63b507729e0a",
		"dbff5c5ac498571abd74ce3bf2cb8e70e06bfa12",
		"039e550dde9aaa94d96c7a4cb2949cb3551b07e0",
		"543dfd7437436cdc173bff5e5247e0f7a2be706d",
		"ce6a0f29e68c23b0350cb5c85f6da344c250ac33",
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"77665b88ab57e56b3a53bc02f7d042499184c320",
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"798486e8b43bb20ef123b628b436fee8cfd2372b",
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
	}
	cmplists(t, paths, expectedPaths, true)
	cmplists(t, perms, expectedPerms, false)
	cmplists(t, shas, expectedShas, false)
}

func TestCopyFile(t *testing.T) {

}
