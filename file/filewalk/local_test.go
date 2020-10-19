package filewalk_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
)

var localTestTree string

func TestMain(m *testing.M) {
	tmpDir, _ := ioutil.TempDir("", "filewalk")
	defer os.RemoveAll(tmpDir)
	if err := createTestDir(tmpDir); err != nil {
		fmt.Printf("failed to create testdir %v", err)
		os.Exit(0)
	}
	localTestTree = tmpDir
	code := m.Run()
	os.Exit(code)
}

func scan(sc filewalk.Filesystem, ch chan filewalk.Contents, dir string) (dirNames, fileNames []string, errors []error, info map[string]filewalk.Info) {
	ctx := context.Background()
	info = map[string]filewalk.Info{}
	go func() {
		sc.List(ctx, dir, ch)
		close(ch)
	}()
	for c := range ch {
		for _, child := range c.Children {
			dirNames = append(dirNames, child.Name)
			info[child.Name] = child
		}
		for _, file := range c.Files {
			fileNames = append(fileNames, file.Name)
			info[file.Name] = file
		}
		if c.Err != nil {
			errors = append(errors, c.Err)
		}
	}
	sort.Strings(dirNames)
	sort.Strings(fileNames)
	return
}

func TestLocalFilesystem(t *testing.T) {
	sc := filewalk.LocalFilesystem(1)
	ch := make(chan filewalk.Contents, 1)

	dirs, files, errors, info := scan(sc, ch, localTestTree)

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
		if _, ok := info[d].Sys().(os.FileInfo); !ok {
			t.Errorf("%v: wrong type for Sys %T", d, info[d].Sys)
		}
		if got, want := info[d].IsPrefix(), true; got != want {
			t.Errorf("%v: got %v, want %v", d, got, want)
		}
		if got, want := info[d].IsLink(), false; got != want {
			t.Errorf("%v: got %v, want %v", d, got, want)
		}
	}

	for _, f := range expectedFileNames {
		if got, want := info[f].IsPrefix(), false; got != want {
			t.Errorf("%v: got %v, want %v", f, got, want)
		}
		if !strings.HasPrefix(f, "l") {
			if got, want := info[f].Size, int64(3); got != want {
				t.Errorf("%v: got %v, want %v", f, got, want)
			}
		}
		if got, want := info[f].IsLink(), strings.HasPrefix(f, "l"); got != want {
			t.Errorf("%v: got %v, want %v", f, got, want)
		}
	}

	ch = make(chan filewalk.Contents)

	dirs, files, errors, info = scan(sc, ch, sc.Join(localTestTree, "inaccessible-dir"))

	if got, want := len(errors), 1; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	if got, want := sc.IsPermissionError(errors[0]), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func createTestDir(tmpDir string) error {
	j := filepath.Join
	errs := errors.M{}
	dirs := []string{
		j("a0"),
		j("a0", "a0.0"),
		j("a0", "a0.1"),
		j("b0", "b0.0"),
		j("b0", "b0.1", "b1.0"),
	}
	for _, dir := range append([]string{""}, dirs...) {
		err := os.MkdirAll(j(tmpDir, dir), 0777)
		errs.Append(err)
		for _, file := range []string{"f0", "f1", "f2"} {
			err = ioutil.WriteFile(j(tmpDir, dir, file), []byte{'1', '2', '3'}, 0666)
			errs.Append(err)
		}
	}
	err := os.Mkdir(j(tmpDir, "inaccessible-dir"), 0000)
	errs.Append(err)
	err = os.Symlink(j("a0", "f0"), j(tmpDir, "lf0"))
	errs.Append(err)
	err = os.Symlink(j("a0"), j(tmpDir, "la0"))
	errs.Append(err)
	err = os.Symlink("nowhere", j(tmpDir, "la1"))
	errs.Append(err)
	err = ioutil.WriteFile(j(tmpDir, "a0", "inaccessible-file"), []byte{'1', '2', '3'}, 0000)
	errs.Append(err)
	return errs.Err()
}
