// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"crypto/sha1"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			fn := strings.TrimPrefix(path, root)
			if isWindows() {
				fn = strings.ReplaceAll(strings.TrimPrefix(path, root), `\`, `/`)
			}
			s := sha1.Sum([]byte(fn))
			shas = append(shas, hex.EncodeToString(s[:]))
			return err
		}
		buf, err := os.ReadFile(path)
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
				t.Errorf("%v: %v: got %v does not have suffix %v", errors.Caller(2, 1), i, got, want)
			}
			continue
		}
		if got, want := a[i], b[i]; got != want {
			t.Errorf("%v: %v: got %v, want %v", errors.Caller(2, 1), i, got, want)
		}
	}
}

func TestMirrorDirTree(t *testing.T) {
	td, err := os.MkdirTemp("", "test-mirror")
	t.Logf("testdir: %v", td)
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	defer os.RemoveAll(td)

	fp := func(p string) string {
		return strings.ReplaceAll(p, "/", string(filepath.Separator))
	}

	expectedPaths := []string{
		fp("testdata"),
		fp("testdata/a"),
		fp("testdata/a/b"),
		fp("testdata/a/b/c"),
		fp("testdata/a/b/c/f5"),
		fp("testdata/a/b/c/f6"),
		fp("testdata/a/b/f3"),
		fp("testdata/a/b/f4"),
		fp("testdata/a/d"),
		fp("testdata/a/d/f7"),
		fp("testdata/a/f1"),
		fp("testdata/a/f2"),
	}
	expectedRegular := []string{
		fp("testdata/a/b/c/f5"),
		fp("testdata/a/b/c/f6"),
		fp("testdata/a/b/f3"),
		fp("testdata/a/b/f4"),
		fp("testdata/a/d/f7"),
		fp("testdata/a/f1"),
		fp("testdata/a/f2"),
	}
	expectedDirs := []string{
		fp("testdata"),
		fp("testdata/a"),
		fp("testdata/a/b"),
		fp("testdata/a/b/c"),
		fp("testdata/a/d"),
	}
	expectedPerms := []string{
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rwxr-xr-x",
		"-rw-r--r--",
		"-rw-r--r--",
		"-rw-r--r--",
		"-rw-r--r--",
		"-rwxr-xr-x",
		"-rw-r--r--",
		"-rw-r--r--",
		"-rw-r--r--",
	}
	if isWindows() {
		expectedPerms = []string{
			"-rwxrwxrwx",
			"-rwxrwxrwx",
			"-rwxrwxrwx",
			"-rwxrwxrwx",
			"-rw-rw-rw-",
			"-rw-rw-rw-",
			"-rw-rw-rw-",
			"-rw-rw-rw-",
			"-rwxrwxrwx",
			"-rw-rw-rw-",
			"-rw-rw-rw-",
			"-rw-rw-rw-",
		}
	}
	expectedShas := []string{
		"7956815567b5ab861b991832a34b63b507729e0a", // sha of filename
		"dbff5c5ac498571abd74ce3bf2cb8e70e06bfa12",
		"039e550dde9aaa94d96c7a4cb2949cb3551b07e0",
		"543dfd7437436cdc173bff5e5247e0f7a2be706d",
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"798486e8b43bb20ef123b628b436fee8cfd2372b",
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // empty file.
	}

	copyall := func(from, suffix, listDir string) (paths, perms, shas []string) {
		todir := filepath.Join(td, suffix)
		if err := os.MkdirAll(todir, 0777); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		if err := cmdutil.CopyAll(from, todir, false); err != nil {
			t.Fatalf("MirrorDirTree: %v", err)
		}
		return list(t, filepath.Join(td, listDir))
	}

	paths, perms, shas := copyall("testdata", "T1", "T1")
	cmplists(t, paths, expectedPaths, true)
	cmplists(t, perms, expectedPerms, false)
	cmplists(t, shas, expectedShas, false)

	regular, err := cmdutil.ListRegular(filepath.Join(td, "T1"))
	if err != nil {
		t.Fatalf("ListRegular: %v", err)
	}

	cmplists(t, regular, expectedRegular, true)

	dirs, err := cmdutil.ListDir(filepath.Join(td, "T1"))
	if err != nil {
		t.Fatalf("ListRegular: %v", err)
	}
	cmplists(t, dirs, expectedDirs, true)
	paths, perms, shas = copyall("testdata/", "T2/testdata", "T2")
	cmplists(t, paths, expectedPaths, true)
	cmplists(t, perms, expectedPerms, false)
	cmplists(t, shas, expectedShas, false)

	err = cmdutil.CopyAll("notadirectory", "testdata", false)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("missing or wrong error: %v", err)
	}
	err = cmdutil.CopyAll("testdata", "notadirectory", false)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("missing or wrong error: %v", err)
	}
	err = cmdutil.CopyAll("testdata", filepath.Join(td, "T1"), false)
	if err == nil || !strings.Contains(err.Error(), "will not overwrite existing file") {
		t.Errorf("missing or wrong error: %v", err)
	}
}

func randContents(t *testing.T, n int) []byte {
	src := rand.NewSource(time.Now().Unix())
	rnd := rand.New(src)
	buf := make([]byte, n)
	_, err := rnd.Read(buf)
	if err != nil {
		t.Fatalf("math.Rand: %v", err)
	}
	return buf
}

func TestCopyFile(t *testing.T) {
	td, err := os.MkdirTemp("", "test-mirror")
	t.Logf("testdir: %v", td)
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	defer os.RemoveAll(td)

	from := filepath.Join(td, "from")
	to := filepath.Join(td, "to")

	newFromFile := func(name string) string {
		buf := randContents(t, 576)
		tmp := sha1.Sum(buf)
		if err := os.WriteFile(name, buf, 0677); err != nil {
			t.Fatalf("failed to create source file")
		}
		return hex.EncodeToString(tmp[:])
	}
	fromSha := newFromFile(from)

	assert := func(err error) {
		if err != nil {
			t.Fatalf("%v: %v", errors.Caller(2, 1), err)
		}
	}

	assertErr := func(err error, possibleErrors ...string) {
		if err == nil {
			t.Errorf("%v: missing error", errors.Caller(2, 1))
			return
		}
		for _, text := range possibleErrors {
			if strings.Contains(err.Error(), text) {
				return
			}
		}
		t.Errorf("%v: missing or wrong error: got %v does not contain any of %q", errors.Caller(2, 1), err, strings.Join(possibleErrors, ", "))
	}

	assert(cmdutil.CopyFile(from, to, 0477, false))

	expectedPaths := []string{"from", "to"}
	expectedShas := []string{fromSha, fromSha}
	expectedPerms := []string{"-rw-r-xr-x", "-r--rwxrwx"}
	if isWindows() {
		expectedPerms = []string{"-rw-rw-rw-", "-r--r--r--"}
	}
	paths, perms, shas := list(t, td)
	cmplists(t, paths, expectedPaths, true)
	cmplists(t, perms, expectedPerms, false)
	cmplists(t, shas, expectedShas, false)
	os.Remove(to)

	assert(cmdutil.CopyFile(from, to, 0677, false))

	expectedPerms = []string{"-rw-r-xr-x", "-rw-rwxrwx"}
	if isWindows() {
		expectedPerms = []string{"-rw-rw-rw-", "-rw-rw-rw-"}
	}
	paths, perms, shas = list(t, td)
	cmplists(t, paths, expectedPaths, true)
	cmplists(t, perms, expectedPerms, false)
	cmplists(t, shas, expectedShas, false)

	// Test overwrite.
	fromNew := filepath.Join(td, "from-new")
	fromNewSha := newFromFile(fromNew)

	assert(cmdutil.CopyFile(from, to, 0644, true))

	expectedPaths = []string{from, fromNew, to}
	expectedPerms = []string{"-rw-r-xr-x", "-rw-r-xr-x", "-rw-r--r--"}
	if isWindows() {
		expectedPerms = []string{"-rw-rw-rw-", "-rw-rw-rw-", "-rw-rw-rw-"}
	}
	expectedShas = []string{fromSha, fromNewSha, fromNewSha}
	paths, perms, shas = list(t, td)
	cmplists(t, paths, expectedPaths, true)
	cmplists(t, perms, expectedPerms, false)
	cmplists(t, shas, expectedShas, false)

	// Test errors.
	err = cmdutil.CopyFile(from, to, 0677, false)
	assertErr(err, "will not overwrite existing file")
	err = cmdutil.CopyFile(from, td, 0677, false)
	assertErr(err, "destination is a directory")
	err = cmdutil.CopyFile(from, td, 0677, true)
	assertErr(err, "destination is a directory")

	// No directory permissions.
	forbidden := filepath.Join(td, "forbidden")
	if err := os.MkdirAll(forbidden, 0400); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	forbiddenFile := filepath.Join(forbidden, "test")
	err = cmdutil.CopyFile(from, forbiddenFile, 0677, true)
	if !isWindows() {
		assertErr(err, "permission denied")
	}

	// No file permissions.
	assert(os.Chmod(forbidden, 0777))
	newFromFile(forbiddenFile)
	assert(os.Chmod(forbiddenFile, 0000))
	err = cmdutil.CopyFile(from, forbiddenFile, 0677, true)
	assertErr(err, "permission denied", "Access is denied")
}
