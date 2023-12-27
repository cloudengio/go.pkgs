// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"io/fs"
	"testing"
	"time"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
	"cloudeng.io/file/matcher"
)

type nameOnly struct{}

func (nameOnly) Name() string { return "" }

type modeOnly struct{}

func (modeOnly) Mode() fs.FileMode { return 0 }

type typeOnly struct{}

func (typeOnly) Type() fs.FileMode { return 0 }

type modTimeOnly struct{}

func (modTimeOnly) ModTime() time.Time { return time.Time{} }

type needsStat struct{}

func (nt needsStat) ModTime() time.Time { return time.Time{} }
func (nt needsStat) Mode() fs.FileMode  { return 0 }

type dirSize struct{}

func (dirSize) NumEntries() int64 { return 0 }

func runParser(t *testing.T, input string) boolexpr.T {
	t.Helper()
	p := matcher.New()
	m, err := p.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestNeedsOps(t *testing.T) {
	var expr boolexpr.T
	assert := func(needsName, needsType, needsMode, needsTime, needsDirSize bool) {
		t.Helper()
		if got, want := expr.Needs(nameOnly{}), needsName; got != want {
			t.Errorf("nameOnly: got %v, want %v", got, want)
		}
		if got, want := expr.Needs(modeOnly{}), needsMode; got != want {
			t.Errorf("modeOnly: got %v, want %v", got, want)
		}
		if got, want := expr.Needs(typeOnly{}), needsType; got != want {
			t.Errorf("typeOnly: got %v, want %v", got, want)
		}
		if got, want := expr.Needs(modTimeOnly{}), needsTime; got != want {
			t.Errorf("time: got %v, want %v", got, want)
		}
		if got, want := expr.Needs(needsStat{}), needsMode || needsTime; got != want {
			t.Errorf("time: got %v, want %v", got, want)
		}
		if got, want := expr.Needs(dirSize{}), needsDirSize; got != want {
			t.Errorf("dirSize: got %v, want %v", got, want)
		}
	}

	expr = runParser(t, "")
	assert(false, false, false, false, false)

	expr = runParser(t, "name=foo")
	assert(true, false, false, false, false)

	expr = runParser(t, "type=f || type=d || type=l")
	assert(false, true, false, false, false)
	expr = runParser(t, "name=xx || type=x || type=f")
	assert(true, true, true, false, false)
	expr = runParser(t, "name=xx || newer=2022-12-12")
	assert(true, false, false, true, false)
	expr = runParser(t, "name=xx || type=d || newer=2022-12-12")
	assert(true, true, false, true, false)

	expr = runParser(t, "name=a && ( name=xx || type=f )")
	assert(true, true, false, false, false)
	expr = runParser(t, "name=a && (name=xx || newer=2022-12-12 )")
	assert(true, false, false, true, false)
	expr = runParser(t, "name=a && (name=xx || type=d || newer=2022-12-12 )")
	assert(true, true, false, true, false)
	expr = runParser(t, "(type=x || newer=2022-12-12 )")
	assert(false, false, true, true, false)
	expr = runParser(t, "(type=x || dir-larger=10)")
	assert(false, false, true, false, true)
	expr = runParser(t, "(type=x || dir-smaller=10)")
	assert(false, false, true, false, true)
}

type dirEntry struct {
	name string
	mode fs.FileMode
}

func (de dirEntry) Name() string      { return de.name }
func (de dirEntry) Path() string      { return de.name }
func (de dirEntry) Type() fs.FileMode { return de.mode }
func (de dirEntry) IsDir() bool       { return de.mode.IsDir() }
func (de dirEntry) Info() (fs.FileInfo, error) {
	return file.NewInfo(de.name, 0, de.mode, time.Time{}, nil), nil
}

type fileInfo struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
	size    int64
}

func (fi fileInfo) Name() string       { return fi.name }
func (fi fileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi fileInfo) ModTime() time.Time { return fi.modTime }
func (fi fileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fileInfo) Size() int64        { return fi.size }
func (fi fileInfo) Info() (fs.FileInfo, error) {
	return file.NewInfo(fi.name, 0, fi.mode, fi.modTime, nil), nil
}

type dirsize struct {
	n int64
}

func (dc dirsize) NumEntries() int64 { return dc.n }

func TestFileOperands(t *testing.T) {
	before, err := time.Parse(time.DateOnly, "2000-01-01")
	if err != nil {
		t.Fatal(err)
	}
	after, err := time.Parse(time.DateOnly, "2020-01-01")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		it     boolexpr.Operand
		de     any
		result bool
	}{
		{matcher.Glob("", "foo", false), dirEntry{name: "foo"}, true},
		{matcher.Glob("", "foo", false), dirEntry{name: "Foo"}, false},
		{matcher.Glob("", "foo", true), dirEntry{name: "Foo"}, true},
		{matcher.Glob("", "f*", true), dirEntry{name: "Fxx"}, true},
		{matcher.Glob("", "x*", false), dirEntry{name: "Fxx"}, false},
		{matcher.Regexp("", "^foo$"), dirEntry{name: "foo"}, true},
		{matcher.Regexp("", "^foo$"), dirEntry{name: "foox"}, false},
		{matcher.FileType("", "f"), dirEntry{name: "foo", mode: 0}, true},
		{matcher.FileType("", "f"), dirEntry{name: "foo", mode: fs.ModeDir}, false},
		{matcher.FileType("", "d"), dirEntry{name: "foo", mode: fs.ModeDir}, true},
		{matcher.FileType("", "l"), dirEntry{name: "foo", mode: fs.ModeDir}, false},
		{matcher.FileType("", "l"), dirEntry{name: "foo", mode: fs.ModeSymlink}, true},
		{matcher.FileType("", "x"), dirEntry{name: "foo", mode: fs.ModeDir | 0111}, false},
		{matcher.FileType("", "x"), dirEntry{name: "foo", mode: fs.ModeSymlink | 0111}, false},
		{matcher.FileType("", "x"), dirEntry{name: "foo", mode: 0111}, false},
		{matcher.NewerThanParsed("", "2010-01-01"), dirEntry{name: "foo"}, false},
		{matcher.FileType("", "f"), fileInfo{mode: 0}, true},
		{matcher.FileType("", "f"), fileInfo{mode: fs.ModeDir}, false},
		{matcher.FileType("", "d"), fileInfo{mode: fs.ModeDir}, true},
		{matcher.FileType("", "l"), fileInfo{mode: fs.ModeDir}, false},
		{matcher.FileType("", "l"), fileInfo{mode: fs.ModeSymlink}, true},
		{matcher.FileType("", "x"), fileInfo{mode: fs.ModeDir | 0111}, false},
		{matcher.FileType("", "x"), fileInfo{mode: fs.ModeSymlink | 0111}, false},
		{matcher.FileType("", "x"), fileInfo{mode: 0111}, true},
		{matcher.NewerThanParsed("", "2010-01-01"), fileInfo{modTime: before}, false},
		{matcher.NewerThanParsed("", "2010-01-01"), fileInfo{modTime: after}, true},
		{matcher.DirSizeLarger("", "100"), dirsize{101}, true},
		{matcher.DirSizeLarger("", "100"), dirsize{99}, false},
		{matcher.DirSizeSmaller("", "100"), dirsize{101}, false},
		{matcher.DirSizeSmaller("", "100"), dirsize{99}, true},
		{matcher.FileSizeLarger("", "100"), fileInfo{size: 101}, true},
		{matcher.FileSizeSmaller("", "100"), fileInfo{size: 100}, true},
	} {
		expr, err := boolexpr.New(boolexpr.NewOperandItem(tc.it))
		if err != nil {
			t.Errorf("%v: failed to create expression: %v", tc.it, err)
			continue
		}
		if got, want := expr.Eval(tc.de), tc.result; got != want {
			t.Errorf("%v: got %v, want %v", tc.it, got, want)
		}
	}
}

type dirEntryPath struct {
	name string
	path string
}

func (de dirEntryPath) Name() string { return de.name }
func (de dirEntryPath) Path() string { return de.path }

func TestNameAndPath(t *testing.T) {
	for _, tc := range []struct {
		it     boolexpr.Operand
		de     any
		result bool
	}{
		{matcher.Glob("", "bar", false), dirEntryPath{name: "bar", path: "foo/bar"}, true},
		{matcher.Glob("", "foo/bar", false), dirEntryPath{name: "bar", path: "foo/bar"}, true},
		{matcher.Glob("", "*/bar", false), dirEntryPath{name: "bar", path: "foo/bar"}, true},
		{matcher.Glob("", "foo/*", false), dirEntryPath{name: "bar", path: "foo/bar"}, true},
		{matcher.Glob("", "*/foo/bar", false), dirEntryPath{name: "bar", path: "foo/bar"}, false},
	} {
		expr, err := boolexpr.New(boolexpr.NewOperandItem(tc.it))
		if err != nil {
			t.Errorf("%v: failed to create expression: %v", tc.it, err)
			continue
		}
		if got, want := expr.Eval(tc.de), tc.result; got != want {
			t.Errorf("%v: got %v, want %v", tc.it, got, want)
		}
	}

}
