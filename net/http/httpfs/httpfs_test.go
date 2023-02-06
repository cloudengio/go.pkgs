// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs_test

import (
	"embed"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

	fetch := func(name string) string {
		f, err := hfs.Open(srv.URL + name)
		if err != nil {
			t.Fatal(err)
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}
		return string(buf)
	}

	buf := fetch("/testdata/a.html")
	if got, want := string(buf), `<html>
<title>A</title>
</html>
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	buf = fetch("/testdata/b")
	if got, want := string(buf), `just a file
`; got != want {
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
	if err == nil || err.Error() != "unsupported scheme: http" {
		t.Fatal(err)
	}

	hfs = httpfs.New(client, httpfs.WithHTTPScheme())
	if got, want := hfs.Scheme(), "http"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, err = hfs.Open("https://foo")
	if err == nil || err.Error() != "unsupported scheme: https" {
		t.Fatal(err)
	}

}
