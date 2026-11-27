// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"cloudeng.io/file/largefile"

	"cloudeng.io/net/http/httpfs"
)

func TestDownloadFile(t *testing.T) {
	const fileContent = "abcdefghijklmnopqrstuvwxyz0123456789"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
			w.WriteHeader(http.StatusOK)
		case "GET":
			rangeHeader := r.Header.Get("Range")
			if !strings.HasPrefix(rangeHeader, "bytes=") {
				http.Error(w, "missing or invalid Range header", http.StatusBadRequest)
				return
			}
			rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
			parts := strings.Split(rangeSpec, "-")
			if len(parts) != 2 {
				http.Error(w, "invalid Range header", http.StatusBadRequest)
				return
			}
			from, err := strconv.Atoi(parts[0])
			if err != nil {
				http.Error(w, "invalid Range header", http.StatusBadRequest)
				return
			}
			to, err := strconv.Atoi(parts[1])
			if err != nil {
				http.Error(w, "invalid Range header", http.StatusBadRequest)
				return
			}
			if from < 0 || to >= len(fileContent) || from > to {
				http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.Header().Set("Content-Range", "bytes "+rangeSpec+"/"+strconv.Itoa(len(fileContent)))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fileContent[from : to+1])) //nolint:errcheck
		default:
			http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		}
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "downloaded_file")

	ctx := context.Background()
	dl := httpfs.NewDownloader()

	// Use a small block size to force multiple requests
	dl.WithReaderOptions(httpfs.WithLargeFileBlockSize(10))

	n, err := dl.DownloadFile(ctx, ts.URL, dest)
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	if n != int64(len(fileContent)) {
		t.Errorf("DownloadFile returned size %d, want %d", n, len(fileContent))
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != fileContent {
		t.Errorf("Downloaded content = %q, want %q", string(data), fileContent)
	}
}

func TestDownloadFileResume(t *testing.T) {
	const content = "abcdefghijklmnopqrstuvwxyz0123456789"
	var failing atomic.Bool
	failing.Store(true)
	var gets atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			return
		}
		var from, to int
		if _, err := fmt.Sscanf(r.Header.Get("Range"), "bytes=%d-%d", &from, &to); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if failing.Load() && from >= 20 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		gets.Add(1)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, to, len(content)))
		w.Write([]byte(content[from : to+1])) //nolint:errcheck
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "file")
	dl := httpfs.NewDownloader().
		WithResume().
		WithReaderOptions(httpfs.WithLargeFileBlockSize(10)).
		WithDownloaderOptions(largefile.WithDownloadConcurrency(1))
	if _, err := dl.DownloadFile(t.Context(), ts.URL, dest); err == nil {
		t.Fatal("expected a failure")
	}
	if _, err := os.Stat(dest + ".partial"); err != nil {
		t.Fatalf("partial download was not retained: %v", err)
	}
	failing.Store(false)
	gets.Store(0)
	n, err := dl.DownloadFile(t.Context(), ts.URL, dest)
	if err != nil || n != int64(len(content)) {
		t.Fatalf("n=%v: %v", n, err)
	}
	if got := gets.Load(); got != 2 {
		t.Errorf("resumed with %d requests, want 2", got)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != content {
		t.Errorf("got %q", data)
	}
}
