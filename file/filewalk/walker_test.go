package filewalk_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk"
)

type logger struct {
	prefix string
	out    *strings.Builder
	skip   string
}

func (l *logger) filesFunc(ctx context.Context, prefix string, ch <-chan filewalk.Contents) error {
	sizes := map[string]int64{}
	files := []string{}
	prefix = strings.TrimPrefix(prefix, l.prefix)
	for results := range ch {
		results.Path = strings.TrimPrefix(results.Path, l.prefix)
		if err := results.Err; err != nil {
			fmt.Fprintf(l.out, "list : %v: %v\n", results.Path,
				strings.Replace(results.Err.Error(), l.prefix, "", 1))
			continue
		}
		for _, info := range results.Files {
			full := filepath.Join(prefix, info.Name)
			files = append(files, full)
			sizes[full] = info.Size
		}
	}
	sort.Strings(files)
	for _, f := range files {
		fmt.Fprintf(l.out, "file : %v: %v\n", f, sizes[f])
	}
	return nil
}

func (l *logger) dirsFunc(ctx context.Context, prefix string, info *filewalk.Info, err error) (bool, error) {
	if err != nil {
		fmt.Fprintf(l.out, "dir  : error: %v: %v\n", prefix, err)
		return true, nil
	}
	prefix = strings.TrimPrefix(prefix, l.prefix)
	if len(l.skip) > 0 && prefix == l.skip {
		fmt.Fprintf(l.out, "skip  : %v: %v\n", prefix, info.Size)
		return true, nil
	}
	fmt.Fprintf(l.out, "dir  : %v: %v\n", prefix, info.Size)
	return false, nil
}

func TestSimple(t *testing.T) {
	tmpDir := t.TempDir()
	t.Log(tmpDir)
	if err := createTestDir(tmpDir); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	sc := filewalk.LocalScanner()

	wk := filewalk.New(sc)
	lg := &logger{out: &strings.Builder{}, prefix: tmpDir}
	testSimple(t, ctx, tmpDir, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.ScanSize(1), filewalk.Concurrency(1))
	lg = &logger{out: &strings.Builder{}, prefix: tmpDir}
	testSimple(t, ctx, tmpDir, wk, lg, expectedFull)

	wk = filewalk.New(sc, filewalk.ScanSize(1), filewalk.Concurrency(1))
	lg = &logger{out: &strings.Builder{}, prefix: tmpDir, skip: "/b0/b0.1"}
	testSimple(t, ctx, tmpDir, wk, lg, expectedPartial1)

	lg = &logger{out: &strings.Builder{}, prefix: tmpDir, skip: "/b0"}
	testSimple(t, ctx, tmpDir, wk, lg, expectedPartial2)
}

func testSimple(t *testing.T, ctx context.Context, tmpDir string, wk *filewalk.Walker, lg *logger, expected string) {
	_, _, line, _ := runtime.Caller(1)
	err := wk.Walk(ctx, lg.dirsFunc, lg.filesFunc, tmpDir)
	if err != nil {
		t.Errorf("line: %v: errors: %v", line, err)
	}

	if got, want := lg.out.String(), expected; got != want {
		t.Errorf("ine: %v: got %v, want %v", line, got, want)
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
	for _, dir := range dirs {
		err := os.MkdirAll(j(tmpDir, dir), 0777)
		errs.Append(err)
		for _, file := range []string{"f0", "f1", "f2"} {
			err = ioutil.WriteFile(j(tmpDir, dir, file), []byte{'1', '2', '3'}, 0666)
			errs.Append(err)
		}
	}
	err := os.Mkdir(j(tmpDir, "inaccessible-dir"), 0000)
	errs.Append(err)
	err = os.Symlink(j(tmpDir, "a0", "f0"), j(tmpDir, "lf0"))
	errs.Append(err)
	err = os.Symlink(j(tmpDir, "a0"), j(tmpDir, "la0"))
	errs.Append(err)
	err = os.Symlink("nowhere", j(tmpDir, "la1"))
	errs.Append(err)
	err = ioutil.WriteFile(j(tmpDir, "a0", "inaccessible-file"), []byte{'1', '2', '3'}, 0000)
	errs.Append(err)
	return errs.Err()
}

const expectedFull = `dir  : : 256
file : la0: 75
file : la1: 7
file : lf0: 78
dir  : /a0: 256
file : /a0/f0: 3
file : /a0/f1: 3
file : /a0/f2: 3
file : /a0/inaccessible-file: 3
dir  : /a0/a0.0: 160
file : /a0/a0.0/f0: 3
file : /a0/a0.0/f1: 3
file : /a0/a0.0/f2: 3
dir  : /a0/a0.1: 160
file : /a0/a0.1/f0: 3
file : /a0/a0.1/f1: 3
file : /a0/a0.1/f2: 3
dir  : /b0: 128
dir  : /b0/b0.0: 160
file : /b0/b0.0/f0: 3
file : /b0/b0.0/f1: 3
file : /b0/b0.0/f2: 3
dir  : /b0/b0.1: 96
dir  : /b0/b0.1/b1.0: 160
file : /b0/b0.1/b1.0/f0: 3
file : /b0/b0.1/b1.0/f1: 3
file : /b0/b0.1/b1.0/f2: 3
dir  : /inaccessible-dir: 64
list : /inaccessible-dir: open /inaccessible-dir: permission denied
`

// No b0.1 sub dir.
const expectedPartial1 = `dir  : : 256
file : la0: 75
file : la1: 7
file : lf0: 78
dir  : /a0: 256
file : /a0/f0: 3
file : /a0/f1: 3
file : /a0/f2: 3
file : /a0/inaccessible-file: 3
dir  : /a0/a0.0: 160
file : /a0/a0.0/f0: 3
file : /a0/a0.0/f1: 3
file : /a0/a0.0/f2: 3
dir  : /a0/a0.1: 160
file : /a0/a0.1/f0: 3
file : /a0/a0.1/f1: 3
file : /a0/a0.1/f2: 3
dir  : /b0: 128
dir  : /b0/b0.0: 160
file : /b0/b0.0/f0: 3
file : /b0/b0.0/f1: 3
file : /b0/b0.0/f2: 3
skip  : /b0/b0.1: 96
dir  : /inaccessible-dir: 64
list : /inaccessible-dir: open /inaccessible-dir: permission denied
`

// No b0 sub dir.
const expectedPartial2 = `dir  : : 256
file : la0: 75
file : la1: 7
file : lf0: 78
dir  : /a0: 256
file : /a0/f0: 3
file : /a0/f1: 3
file : /a0/f2: 3
file : /a0/inaccessible-file: 3
dir  : /a0/a0.0: 160
file : /a0/a0.0/f0: 3
file : /a0/a0.0/f1: 3
file : /a0/a0.0/f2: 3
dir  : /a0/a0.1: 160
file : /a0/a0.1/f0: 3
file : /a0/a0.1/f1: 3
file : /a0/a0.1/f2: 3
skip  : /b0: 128
dir  : /inaccessible-dir: 64
list : /inaccessible-dir: open /inaccessible-dir: permission denied
`
