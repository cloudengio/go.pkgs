// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"io/fs"
	"testing"
	"time"

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

func TestNeedsOps(t *testing.T) {
	var err error
	assert := func(m matcher.T, needsName, needsType, needsMode, needsTime bool) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := m.Needs(nameOnly{}), needsName; got != want {
			t.Errorf("nameOnly: got %v, want %v", got, want)
		}
		if got, want := m.Needs(modeOnly{}), needsMode; got != want {
			t.Errorf("modeOnly: got %v, want %v", got, want)
		}
		if got, want := m.Needs(typeOnly{}), needsType; got != want {
			t.Errorf("typeOnly: got %v, want %v", got, want)
		}
		if got, want := m.Needs(modTimeOnly{}), needsTime; got != want {
			t.Errorf("time: got %v, want %v", got, want)
		}
		if got, want := m.Needs(needsStat{}), needsMode || needsTime; got != want {
			t.Errorf("time: got %v, want %v", got, want)
		}
	}

	var m matcher.T
	assert(m, false, false, false, false)
	if got, want := m.Needs(file.Info{}), false; got != want {
		t.Errorf("file.Info: got %v, want %v", got, want)
	}

	m, err = matcher.New(parse("xx")...)
	if err != nil {
		t.Fatal(err)
	}
	assert(m, true, false, false, false)

	m, err = matcher.New(parse("xx || ft: f || ft: d || ft: l")...)
	assert(m, true, true, false, false)
	m, err = matcher.New(parse("xx || ft: x || ft: f")...)
	assert(m, true, true, true, false)
	m, err = matcher.New(parse("xx || nt: 2022-12-12 :nt")...)
	assert(m, true, false, false, true)
	m, err = matcher.New(parse("xx || ft: d || nt: 2022-12-12 :nt")...)
	assert(m, true, true, false, true)

	m, err = matcher.New(parse("a && ( xx || ft: f )")...)
	assert(m, true, true, false, false)
	m, err = matcher.New(parse("a && (xx || nt: 2022-12-12 :nt )")...)
	assert(m, true, false, false, true)
	m, err = matcher.New(parse("a && (xx || ft: d || nt: 2022-12-12 :nt )")...)
	assert(m, true, true, false, true)
	m, err = matcher.New(parse(" (ft: x || nt: 2022-12-12 :nt )")...)
	assert(m, false, false, true, true)

	if got, want := m.Needs(file.Info{}), true; got != want {
		t.Errorf("file.Info: got %v, want %v", got, want)
	}
}

type dirEntry struct {
	name string
	mode fs.FileMode
}

func (de dirEntry) Name() string      { return de.name }
func (de dirEntry) Type() fs.FileMode { return de.mode }
func (de dirEntry) IsDir() bool       { return de.mode.IsDir() }
func (de dirEntry) Info() (fs.FileInfo, error) {
	return file.NewInfo(de.name, 0, de.mode, time.Time{}, nil), nil
}

type fileInfo struct {
	name    string
	mode    fs.FileMode
	modTime time.Time
}

func (fi fileInfo) Name() string       { return fi.name }
func (fi fileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi fileInfo) ModTime() time.Time { return fi.modTime }
func (fi fileInfo) IsDir() bool        { return fi.mode.IsDir() }
func (fi fileInfo) Info() (fs.FileInfo, error) {
	return file.NewInfo(fi.name, 0, fi.mode, fi.modTime, nil), nil
}

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
		it     matcher.Item
		de     any
		result bool
	}{
		{matcher.Glob("foo", false), dirEntry{name: "foo"}, true},
		{matcher.Glob("foo", false), dirEntry{name: "Foo"}, false},
		{matcher.Glob("foo", true), dirEntry{name: "Foo"}, true},
		{matcher.Glob("f*", true), dirEntry{name: "Fxx"}, true},
		{matcher.Glob("x*", false), dirEntry{name: "Fxx"}, false},
		{matcher.Regexp("^foo$"), dirEntry{name: "foo"}, true},
		{matcher.Regexp("^foo$"), dirEntry{name: "foox"}, false},
		{matcher.FileType("f"), dirEntry{name: "foo", mode: 0}, true},
		{matcher.FileType("f"), dirEntry{name: "foo", mode: fs.ModeDir}, false},
		{matcher.FileType("d"), dirEntry{name: "foo", mode: fs.ModeDir}, true},
		{matcher.FileType("l"), dirEntry{name: "foo", mode: fs.ModeDir}, false},
		{matcher.FileType("l"), dirEntry{name: "foo", mode: fs.ModeSymlink}, true},
		{matcher.FileType("x"), dirEntry{name: "foo", mode: fs.ModeDir | 0111}, false},
		{matcher.FileType("x"), dirEntry{name: "foo", mode: fs.ModeSymlink | 0111}, false},
		{matcher.FileType("x"), dirEntry{name: "foo", mode: 0111}, false},
		{matcher.NewerThanParsed("2010-01-01"), dirEntry{name: "foo"}, false},

		{matcher.FileType("f"), fileInfo{mode: 0}, true},
		{matcher.FileType("f"), fileInfo{mode: fs.ModeDir}, false},
		{matcher.FileType("d"), fileInfo{mode: fs.ModeDir}, true},
		{matcher.FileType("l"), fileInfo{mode: fs.ModeDir}, false},
		{matcher.FileType("l"), fileInfo{mode: fs.ModeSymlink}, true},
		{matcher.FileType("x"), fileInfo{mode: fs.ModeDir | 0111}, false},
		{matcher.FileType("x"), fileInfo{mode: fs.ModeSymlink | 0111}, false},
		{matcher.FileType("x"), fileInfo{mode: 0111}, true},
		{matcher.NewerThanParsed("2010-01-01"), fileInfo{modTime: before}, false},
		{matcher.NewerThanParsed("2010-01-01"), fileInfo{modTime: after}, true},
	} {
		expr, err := matcher.New(tc.it)
		if err != nil {
			t.Errorf("%v: failed to create expression: %v", tc.it, err)
			continue
		}
		if got, want := expr.Eval(tc.de), tc.result; got != want {
			t.Errorf("%v: got %v, want %v", tc.it, got, want)
		}
	}
}
