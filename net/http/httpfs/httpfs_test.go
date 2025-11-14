// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs_test

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudeng.io/file"
	"cloudeng.io/net/http/httperror"
	"cloudeng.io/net/http/httpfs"
)

//go:embed testdata/*
var testdata embed.FS

func TestHTTPFS(t *testing.T) {
	handler := http.FileServer(http.FS(testdata))
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := http.DefaultClient
	hfs := httpfs.New(client, httpfs.WithHTTPScheme())

	fetch := func(name string) (string, fs.FileInfo) {
		f, err := hfs.Open(srv.URL + name)
		if err != nil {
			t.Fatal(err)
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}
		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		return string(buf), fi
	}

	buf, fi := fetch("/testdata/a.html")
	if got, want := buf, `<html>
<title>A</title>
</html>
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fi.Name(), srv.URL+"/testdata/a.html"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	si := fi.Sys().(*httpfs.Response)
	if got, want := si.ContentLength, int64(len(buf)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	buf, fi = fetch("/testdata/b")
	if got, want := buf, `just a file
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	si = fi.Sys().(*httpfs.Response)
	if got, want := si.ContentLength, int64(len(buf)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, err := hfs.Open(srv.URL + "/not there")
	if !httperror.IsHTTPError(err, http.StatusNotFound) {
		t.Fatalf("expected a 404 error, but got %v\n", err)
	}
}

func TestScheme(t *testing.T) {
	client := http.DefaultClient
	hfs := httpfs.New(client)
	if got, want := hfs.Scheme(), "https"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, err := hfs.Open("http://foo")
	if err == nil || !errors.Is(err, file.ErrSchemeNotSupported) {
		t.Fatal(err)
	}

	hfs = httpfs.New(client, httpfs.WithHTTPScheme())
	if got, want := hfs.Scheme(), "http"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, err = hfs.Open("https://foo")
	if err == nil || !errors.Is(err, file.ErrSchemeNotSupported) {
		t.Fatal(err)
	}
}

func TestIsPermissionError(t *testing.T) {
	fs := httpfs.New(http.DefaultClient)

	permErr := &httperror.T{StatusCode: http.StatusForbidden}
	notPermErr := &httperror.T{StatusCode: http.StatusNotFound}
	otherErr := errors.New("some other error")

	if !fs.IsPermissionError(permErr) {
		t.Errorf("expected IsPermissionError to return true for StatusForbidden")
	}
	if fs.IsPermissionError(notPermErr) {
		t.Errorf("expected IsPermissionError to return false for StatusNotFound")
	}
	if fs.IsPermissionError(otherErr) {
		t.Errorf("expected IsPermissionError to return false for unrelated error")
	}
}

func TestIsNotExist(t *testing.T) {
	fs := httpfs.New(http.DefaultClient)

	notExistErr := &httperror.T{StatusCode: http.StatusNotFound}
	permErr := &httperror.T{StatusCode: http.StatusForbidden}
	otherErr := errors.New("some other error")

	if !fs.IsNotExist(notExistErr) {
		t.Errorf("expected IsNotExist to return true for StatusNotFound")
	}
	if fs.IsNotExist(permErr) {
		t.Errorf("expected IsNotExist to return false for StatusForbidden")
	}
	if fs.IsNotExist(otherErr) {
		t.Errorf("expected IsNotExist to return false for unrelated error")
	}
}
