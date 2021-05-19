package webassets_test

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/webapp/webassets"
)

//go:embed testdata/*
var content embed.FS

func createMirror(t *testing.T, tmpDir string) {
	if err := os.MkdirAll(filepath.Join(tmpDir, "testdata", "d0"), 0700); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(tmpDir, "testdata", "hello.txt")
	if err := os.WriteFile(a, []byte("not hello...."), 0600); err != nil {
		t.Fatal(err)
	}
	b := filepath.Join(tmpDir, "testdata", "d0", "world.txt")
	if err := os.WriteFile(b, []byte("not d0/world...."), 0600); err != nil {
		t.Fatal(err)
	}
}

func readFromFS(fs fs.FS, name string) (string, error) {
	f, err := fs.Open(path.Join("testdata", name))
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
			t.Fatalf("failed reading: %v: %v", name, err)
		}
		output = append(output, strings.TrimSpace(o))
	}
	return strings.Join(output, "\n")
}

func TestData(t *testing.T) {
	files := []string{"hello.txt", "world.txt", "d0/hello.txt", "d0/world.txt"}
	contents := readAll(t, content, files...)
	if got, want := contents, `hello
world
d0/hello
d0/world`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	tmpDir := t.TempDir()
	dynamic := webassets.Reloadable(content, tmpDir)
	contents = readAll(t, dynamic, files...)
	if got, want := contents, `hello
world
d0/hello
d0/world`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	createMirror(t, tmpDir)
	contents = readAll(t, dynamic, files...)
	if got, want := contents, `not hello....
world
d0/hello
not d0/world....`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	dynamic = webassets.Reloadable(nil, tmpDir)
	contents = readAll(t, dynamic, "hello.txt", path.Join("d0", "world.txt"))
	if got, want := contents, `not hello....
not d0/world....`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, err := dynamic.Open("world.txt")
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("missing or wrong error: %v", err)
	}
}
