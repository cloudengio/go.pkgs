// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webassets_test

import (
	"embed"
	"io"
	"io/fs"
	"runtime"
	"strings"
	"testing"

	"cloudeng.io/webapp/webassets"
)

//go:embed testdata
var content embed.FS

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

func TestRelative(t *testing.T) {
	files := []string{"hello.txt", "world.txt", "d0/hello.txt", "d0/world.txt"}
	relativeContents := webassets.RelativeFS("testdata", content)
	contents := readAll(t, relativeContents, files...)
	if got, want := contents, `hello
world
d0/hello
d0/world`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
