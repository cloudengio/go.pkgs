// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package gdrive_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloudeng.io/algo/digests"
	"cloudeng.io/google/cloud/gdrive"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	testFileID      = "test-file-id"
	testFileName    = "my-test-file.txt"
	testMD5Checksum = "d41d8cd98f00b204e9800998ecf8427e" // md5 of empty string
)

var testFileContent = []byte("abcdefghijklmnopqrstuvwxyz")

type mockServer struct {
	t            *testing.T
	fileContent  []byte
	fileMetadata *drive.File
	failWithCode int
}

func (m *mockServer) handler(w http.ResponseWriter, r *http.Request) {

	// Metadata request
	if strings.Contains(r.URL.Path, testFileID) && r.URL.Query().Get("alt") != "media" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.fileMetadata)
		return
	}

	if m.failWithCode != 0 {
		http.Error(w, http.StatusText(m.failWithCode), m.failWithCode)
		return
	}

	// Download request
	if strings.Contains(r.URL.Path, testFileID) && r.URL.Query().Get("alt") == "media" {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			http.Error(w, "missing range header", http.StatusBadRequest)
			return
		}

		var from, to int
		_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &from, &to)
		if err != nil {
			http.Error(w, "invalid range header", http.StatusBadRequest)
			return
		}

		if from < 0 || to >= len(m.fileContent) || from > to {
			http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
			return
		}

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", from, to, len(m.fileContent)))
		w.Header().Set("Content-Length", strconv.Itoa(to-from+1))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(m.fileContent[from : to+1])
		return
	}

	http.NotFound(w, r)
}

func newTestServer(t *testing.T) (*httptest.Server, *mockServer) {
	mock := &mockServer{
		t:           t,
		fileContent: testFileContent,
		fileMetadata: &drive.File{
			Id:          testFileID,
			Name:        testFileName,
			Size:        int64(len(testFileContent)),
			Md5Checksum: testMD5Checksum,
		},
	}
	return httptest.NewServer(http.HandlerFunc(mock.handler)), mock
}

func TestDriveReader_New(t *testing.T) {
	ctx := context.Background()
	server, _ := newTestServer(t)
	defer server.Close()

	service, err := drive.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	// Success case
	blockSize := 1234
	dr, err := gdrive.NewReader(ctx, service, testFileID, gdrive.WithBlockSize(blockSize))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	if got, want := dr.Name(), testFileName; got != want {
		t.Errorf("got name %q, want %q", got, want)
	}

	size, bs := dr.ContentLengthAndBlockSize()
	if got, want := size, int64(len(testFileContent)); got != want {
		t.Errorf("got size %v, want %v", got, want)
	}
	if got, want := bs, blockSize; got != want {
		t.Errorf("got blocksize %v, want %v", got, want)
	}

	expectedDigest, _ := digests.New(digests.MD5, []byte{0xd4, 0x1d, 0x8c, 0xd9, 0x8f, 0x00, 0xb2, 0x04, 0xe9, 0x80, 0x09, 0x98, 0xec, 0xf8, 0x42, 0x7e})
	wantDigest := dr.Digest()
	if got, want := wantDigest.Algo, expectedDigest.Algo; got != want {
		t.Errorf("got digest %v, want %v", got, want)
	}
	if got, want := wantDigest.Digest, expectedDigest.Digest; !slices.Equal(got, want) {
		t.Errorf("got digest %v, want %v", got, want)
	}
}

func TestDriveReader_GetReader(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, _ := newTestServer(t)
		defer server.Close()
		service, err := drive.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()))
		if err != nil {
			t.Fatalf("failed to create drive service: %v", err)
		}
		dr, err := gdrive.NewReader(ctx, service, testFileID)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}

		from, to := int64(5), int64(15)
		rc, retry, err := dr.GetReader(ctx, from, to)
		if err != nil {
			t.Fatalf("GetReader failed: %v", err)
		}
		if retry.IsRetryable() {
			t.Error("expected not to be retryable")
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read failed: %v", err)
		}
		if got, want := string(data), string(testFileContent[from:to+1]); got != want {
			t.Errorf("got content %q, want %q", got, want)
		}
	})

	t.Run("retryable error", func(t *testing.T) {
		server, mock := newTestServer(t)
		defer server.Close()
		mock.failWithCode = http.StatusInternalServerError // 500
		service, err := drive.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()))
		if err != nil {
			t.Fatalf("failed to create drive service: %v", err)
		}
		dr, err := gdrive.NewReader(ctx, service, testFileID)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}

		_, retry, err := dr.GetReader(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected an error")
		}
		if !retry.IsRetryable() {
			t.Error("expected to be retryable")
		}
	})

	t.Run("non-retryable error", func(t *testing.T) {
		server, mock := newTestServer(t)
		defer server.Close()
		mock.failWithCode = http.StatusNotFound // 404
		service, _ := drive.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(server.Client()))
		dr, err := gdrive.NewReader(ctx, service, testFileID)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}

		_, retry, err := dr.GetReader(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected an error")
		}
		if retry.IsRetryable() {
			t.Error("expected not to be retryable")
		}
	})
}
