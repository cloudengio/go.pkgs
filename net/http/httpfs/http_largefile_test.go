// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/net/http/httpfs"
	"cloudeng.io/net/http/httpfs/rfc9530"
)

// GenAI: gemini 2.5 generated these tests. It forgot to create an http.Transport
// for the http.Client, so we create a default one here.
func newHTTTransport() *http.Transport {
	transport := &http.Transport{}
	return transport
}

func TestLargeFile_HeadAndRange(t *testing.T) { //nolint:gocyclo
	const fileContent = "abcdefghijklmnopqrstuvwxyz0123456789"
	const blockSize = 8
	const digestOnly = "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="
	const digestVal = "sha-256=:" + digestOnly + ":"

	// Serve HEAD and GET with Accept-Ranges and Repr-Digest headers.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
			w.Header().Set(rfc9530.ReprDigestHeader, digestVal)
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
			from, _ := strconv.Atoi(parts[0])
			to, _ := strconv.Atoi(parts[1])
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

	ctx := context.Background()
	lf, err := httpfs.NewLargeFile(ctx, ts.URL,
		httpfs.WithLargeFileTransport(newHTTTransport()),
		httpfs.WithLargeFileBlockSize(blockSize),
		httpfs.WithLargeFileLogger(slog.Default()),
	)
	if err != nil {
		t.Fatalf("NewLargeFile failed: %v", err)
	}

	if got, want := lf.Name(), ts.URL; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
	cl, bs := lf.ContentLengthAndBlockSize()
	if cl != int64(len(fileContent)) {
		t.Errorf("ContentLength = %d, want %d", cl, len(fileContent))
	}
	if bs != blockSize {
		t.Errorf("BlockSize = %d, want %d", bs, blockSize)
	}
	digest := lf.Digest()
	if got, want := digest.Algo, "sha-256"; got != want {
		t.Errorf("Digest().Algo = %q, want %q", got, want)
	}
	b64 := digests.ToBase64(digest.Digest)
	if got, want := b64, digestOnly; got != want {
		t.Errorf("Digest().Base64 = %q, want %q", got, want)
	}

	// Test reading a block
	from, to := 8, 15
	rc, retry, err := lf.GetReader(ctx, int64(from), int64(to))
	if err != nil {
		t.Fatalf("GetReader failed: %v", err)
	}
	if retry != nil {
		t.Errorf("GetReader returned unexpected retry: %v", retry)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}
	if got, want := string(data), fileContent[from:to+1]; got != want {
		t.Errorf("Read block = %q, want %q", got, want)
	}
}

func TestLargeFile_NoRangeSupport(t *testing.T) {
	// HEAD returns Accept-Ranges: none
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Accept-Ranges", "none")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx := context.Background()
	_, err := httpfs.NewLargeFile(ctx, ts.URL)
	if err == nil || !strings.Contains(err.Error(), "does not support range requests") {
		t.Errorf("expected ErrNoRangeSupport, got %v", err)
	}
}

func TestLargeFile_GetReader_ServiceUnavailableWithRetryAfter(t *testing.T) {
	const fileContent = "abcdefghijklmnopqrstuvwxyz0123456789"
	var getCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
			w.WriteHeader(http.StatusOK)
		case "GET":
			getCount++
			if getCount == 1 {
				w.Header().Set("Retry-After", "2")
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Range", "bytes 0-3/36")
			w.WriteHeader(http.StatusOK) //nolint:errcheck
			w.Write([]byte(fileContent[0:4]))
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	lf, err := httpfs.NewLargeFile(ctx, ts.URL)
	if err != nil {
		t.Fatalf("NewLargeFile failed: %v", err)
	}

	// First GET returns 503 with Retry-After
	rc, retry, err := lf.GetReader(ctx, 0, 3)
	if err != nil {
		t.Fatalf("GetReader failed: %v", err)
	}
	if rc != nil {
		t.Errorf("expected nil ReadCloser on 503")
	}
	if retry == nil {
		t.Fatalf("expected RetryResponse on 503")
	}
	retryable := retry.IsRetryable()
	hasDelay, delay := retry.BackoffDuration()
	if !retryable || !hasDelay || delay != 2*time.Second {
		t.Errorf("unexpected retry info: retryable=%v, hasDelay=%v, delay=%v", retryable, hasDelay, delay)
	}
}

func TestLargeFile_GetReader_BadStatus(t *testing.T) {
	const fileContent = "abc"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
			w.WriteHeader(http.StatusOK)
		case "GET":
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	lf, err := httpfs.NewLargeFile(ctx, ts.URL)
	if err != nil {
		t.Fatalf("NewLargeFile failed: %v", err)
	}

	rc, retry, err := lf.GetReader(ctx, 0, 2)
	if err == nil || !strings.Contains(err.Error(), "bad status code") {
		t.Errorf("expected bad status code error, got %v", err)
	}
	if rc != nil {
		t.Errorf("expected nil ReadCloser on error")
	}
	if retry != nil {
		t.Errorf("expected nil RetryResponse on error")
	}
}

func TestLargeFile_GetReader_InvalidRange(t *testing.T) {
	const fileContent = "abc"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", strconv.Itoa(len(fileContent)))
			w.WriteHeader(http.StatusOK)
		case "GET":
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	lf, err := httpfs.NewLargeFile(ctx, ts.URL)
	if err != nil {
		t.Fatalf("NewLargeFile failed: %v", err)
	}

	rc, retry, err := lf.GetReader(ctx, 10, 20)
	if err == nil || !strings.Contains(err.Error(), "bad status code") {
		t.Errorf("expected bad status code error, got %v", err)
	}
	if rc != nil {
		t.Errorf("expected nil ReadCloser on error")
	}
	if retry != nil {
		t.Errorf("expected nil RetryResponse on error")
	}
}

func TestLargeFile_GetReader_HTTPClientError(t *testing.T) {
	// Use an invalid URL to force a client error
	ctx := context.Background()
	_, err := httpfs.NewLargeFile(ctx, "http://invalid.invalid")
	if err == nil {
		t.Errorf("expected error for invalid URL")
	}
}
