// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package reloadfs_test

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloudeng.io/io/reloadfs"
)

//go:embed testdata
var content embed.FS

func createMirror(t *testing.T, tmpDir string) func() {
	if err := os.MkdirAll(filepath.Join(tmpDir, "testdata", "d0"), 0700); err != nil {
		t.Fatal(err)
	}
	ud := filepath.Join(tmpDir, "testdata", "statwillfail")
	if err := os.MkdirAll(ud, 0700); err != nil {
		t.Fatal(err)
	}
	writeFile := func(name, content string, mode os.FileMode) {
		a := filepath.Join(tmpDir, "testdata", name)
		if err := os.WriteFile(a, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("hello.txt", "not hello....", 0600) // different size.
	writeFile("world.txt", "world\n", 0600)       // same size/contents
	writeFile("a-new-file.txt", "new data...", 0600)
	writeFile(filepath.Join("d0", "world.txt"), "not d0/world....", 0600)

	writeFile(filepath.Join("statwillfail", "statwillfail.txt"), "not hello....", 0600)
	if err := os.Chmod(ud, 000); err != nil {
		t.Fatal(err)
	}

	writeFile("open-will-fail.txt", "can-read-me", 0000)

	return func() {
		os.Chmod(ud, 0700)
	}
}

func readFromFS(fs fs.FS, name string) (string, error) {
	f, err := fs.Open(name)
	if err != nil {
		return "", err
	}
	buf, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func readAll(t *testing.T, fs fs.FS, names ...string) string {
	output := []string{}
	for _, name := range names {
		o, err := readFromFS(fs, name)
		if err != nil {
			_, _, line, _ := runtime.Caller(1)
			t.Fatalf("line: %v: failed reading: %v: %v", line, name, err)
		}
		output = append(output, strings.TrimSpace(o))
	}
	return strings.Join(output, "\n")
}

func TestData(t *testing.T) {
	files := []string{"hello.txt", "world.txt", "d0/hello.txt", "d0/world.txt"}
	tmpDir := t.TempDir()
	dynamic := reloadfs.New(tmpDir, "testdata", content)
	contents := readAll(t, dynamic, files...)
	if got, want := contents, `hello
world
d0/hello
d0/world`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cleanup := createMirror(t, tmpDir)
	defer cleanup()
	contents = readAll(t, dynamic, files...)
	if got, want := contents, `not hello....
world
d0/hello
not d0/world....`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Verify that non-existing or new files are not found.
	for _, name := range []string{"does-not-exist.txt", "a-new-file.txt"} {
		if _, err := dynamic.Open(name); err == nil || !os.IsNotExist(err) {
			t.Errorf("%v: missing or wrong error: %v", name, err)
		}
	}

	dynamic = reloadfs.New(tmpDir, "testdata", content, reloadfs.LoadNewFiles(true))
	if _, err := dynamic.Open("a-new-file.txt"); err != nil {
		t.Errorf("new file should have been found: %v", err)
	}

	contents = readAll(t, dynamic, "hello.txt", path.Join("d0", "world.txt"))
	if got, want := contents, `not hello....
not d0/world....`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := dynamic.Open("a-new-file.txt"); err != nil {
		t.Errorf("new file should have been found: %v", err)
	}

	if _, err := dynamic.Open("non-existent-file.txt"); err == nil || !os.IsNotExist(err) {
		t.Errorf("missing or wrong error: %v", err)
	}

	if _, err := dynamic.Open(path.Join("statwillfail", "statwillfail.txt")); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("missing or wrong error: %v", err)
	}

	if _, err := dynamic.Open("//invalid\\path.txt"); err == nil || !strings.Contains(err.Error(), "invalid argument") {
		t.Errorf("missing or wrong error: %v", err)
	}
}

func TestLogging(t *testing.T) {
	tmpDir := t.TempDir()

	out := &strings.Builder{}
	logger := func(action reloadfs.Action, name, path string, err error) {
		out.WriteString(fmt.Sprintf("%s: %s -> %s: %v", action, name, path, err))
	}

	dynamic := reloadfs.New(tmpDir, "testdata", content, reloadfs.UseLogger(logger))
	dynamic.Open("hello.txt")
	if got, want := out.String(), "reused: hello.txt -> testdata/hello.txt: <nil>"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cleanup := createMirror(t, tmpDir)
	defer cleanup()

	out.Reset()
	dynamic.Open("hello.txt")
	if got, want := out.String(), fmt.Sprintf("reloaded existing: hello.txt -> %s/testdata/hello.txt: <nil>", tmpDir); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out.Reset()
	dynamic.Open("a-new-file.txt")
	if got, want := out.String(), fmt.Sprintf("new files not allowed: a-new-file.txt -> %s/testdata/a-new-file.txt: file does not exist", tmpDir); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	dynamic = reloadfs.New(tmpDir, "testdata", content, reloadfs.UseLogger(logger), reloadfs.LoadNewFiles(true))
	out.Reset()
	dynamic.Open("a-new-file.txt")
	if got, want := out.String(), fmt.Sprintf("reloaded new file: a-new-file.txt -> %s/testdata/a-new-file.txt: <nil>", tmpDir); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestModTime(t *testing.T) {
	tmpDir := t.TempDir()

	out := &strings.Builder{}
	logger := func(action reloadfs.Action, name, path string, err error) {
		out.WriteString(fmt.Sprintf("%s: %s -> %s: %v", action, name, path, err))
	}

	cleanup := createMirror(t, tmpDir)
	defer cleanup()

	dynamic := reloadfs.New(tmpDir, "testdata", content,
		reloadfs.UseLogger(logger),
		reloadfs.ReloadAfter(time.Now().Add(time.Hour)),
	)

	// Sizes differ.
	dynamic.Open("hello.txt")
	if got, want := out.String(), fmt.Sprintf("reloaded existing: hello.txt -> %s/testdata/hello.txt: <nil>", tmpDir); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Same size, but considered too old.
	out.Reset()
	dynamic.Open("world.txt")
	if got, want := out.String(), "reused: world.txt -> testdata/world.txt: <nil>"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
