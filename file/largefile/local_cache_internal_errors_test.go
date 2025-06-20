// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// mockFile implements CacheFileReadWriter to simulate I/O errors for testing.
// It uses a []byte slice for its data store to correctly implement
// io.ReaderAt and io.WriterAt.
type mockFile struct {
	name       string
	readErr    error // Used by Read to simulate io.ReadAll errors
	readAtErr  error
	writeAtErr error
	syncErr    error
	closeErr   error

	shortWrite bool
	data       []byte
	readOffset int64
}

// newMockFile creates a new mock file with optional initial data.
func newMockFile(name string, initialData []byte) *mockFile {
	return &mockFile{
		name: name,
		data: initialData,
	}
}

func (m *mockFile) Read(p []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	if m.readOffset >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n := copy(p, m.data[m.readOffset:])
	m.readOffset += int64(n)
	return n, nil
}

func (m *mockFile) Write(p []byte) (int, error) {
	// For simplicity, Write appends. This is not used by the tests.
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *mockFile) ReadAt(p []byte, off int64) (int, error) {
	if m.readAtErr != nil {
		return 0, m.readAtErr
	}
	if off < 0 {
		return 0, errors.New("readat: negative offset")
	}
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n := copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF // As per io.ReaderAt contract
	}
	return n, nil
}

func (m *mockFile) WriteAt(p []byte, off int64) (int, error) {
	if m.writeAtErr != nil {
		return 0, m.writeAtErr
	}
	if m.shortWrite {
		return len(p) - 1, nil
	}
	if off < 0 {
		return 0, errors.New("writeat: negative offset")
	}
	end := off + int64(len(p))
	if int64(len(m.data)) < end {
		// Grow the slice to accommodate the write.
		newData := make([]byte, end)
		copy(newData, m.data)
		m.data = newData
	}
	n := copy(m.data[off:], p)
	return n, nil
}

func (m *mockFile) Close() error {
	return m.closeErr
}

func (m *mockFile) Sync() error {
	return m.syncErr
}

func (m *mockFile) Name() string {
	return m.name
}

func TestIndexStoreLoadErrors(t *testing.T) {
	mockDataFile := newMockFile("data.dat", nil)

	t.Run("read error", func(t *testing.T) {
		simulatedErr := errors.New("simulated read failure")
		mockIndexFile := newMockFile("index.idx", nil)
		mockIndexFile.readErr = simulatedErr

		_, err := NewLocalDownloadCache(mockDataFile, mockIndexFile)
		if err == nil {
			t.Fatal("NewLocalDownloadCache should have failed but did not")
		}
		if !strings.Contains(err.Error(), "failed to read index file") {
			t.Errorf("error message mismatch: got %v", err)
		}
		if !errors.Is(err, simulatedErr) {
			t.Errorf("expected underlying error to be %v, but it was not found in chain", simulatedErr)
		}
	})

	t.Run("unmarshal error", func(t *testing.T) {
		mockIndexFile := newMockFile("index.idx", []byte("this is not valid json"))
		_, err := NewLocalDownloadCache(mockDataFile, mockIndexFile)
		if err == nil {
			t.Fatal("NewLocalDownloadCache should have failed but did not")
		}
		if !strings.Contains(err.Error(), "failed to unmarshal index file") {
			t.Errorf("error message mismatch: got %v", err)
		}
		var jsonErr *json.SyntaxError
		if !errors.As(err, &jsonErr) {
			t.Errorf("expected a json.SyntaxError, but it was not found in chain")
		}
	})
}

// setupTestCache creates a valid cache with real files and returns the cache instance
// and paths to the files. It's a helper for tests that need a valid starting state.
func setupTestCache(t *testing.T) (*LocalDownloadCache, string, string) {
	t.Helper()
	ctx := context.Background()
	const contentSize int64 = 128
	const blockSize int = 64
	tmpDir := t.TempDir()
	cacheFilePath := filepath.Join(tmpDir, "cache.dat")
	indexFilePath := filepath.Join(tmpDir, "cache.idx")

	if err := CreateNewFilesForCache(ctx, cacheFilePath, indexFilePath, contentSize, blockSize, 1, nil); err != nil {
		t.Fatalf("CreateNewFilesForCache failed: %v", err)
	}
	cacheFile, indexFile, err := OpenCacheFiles(cacheFilePath, indexFilePath)
	if err != nil {
		t.Fatalf("OpenCacheFiles failed: %v", err)
	}
	cache, err := NewLocalDownloadCache(cacheFile, indexFile)
	if err != nil {
		t.Fatalf("NewLocalDownloadCache failed: %v", err)
	}
	return cache, cacheFilePath, indexFilePath
}

func TestInternalCacheErrors(t *testing.T) { //nolint:gocyclo
	t.Run("WriteAt data file error", func(t *testing.T) {
		cache, _, _ := setupTestCache(t)
		defer cache.Close()

		simulatedErr := errors.New("simulated data write error")
		mockDataFile := newMockFile("mock.dat", nil)
		mockDataFile.writeAtErr = simulatedErr
		cache.data = mockDataFile // Replace real file with mock

		_, err := cache.WriteAt(make([]byte, 64), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError, got %T: %v", err, err)
		}
		if !errors.Is(err, simulatedErr) {
			t.Errorf("expected underlying error to be %v, but it was not found in chain", simulatedErr)
		}
	})

	t.Run("WriteAt data sync error", func(t *testing.T) {
		cache, _, _ := setupTestCache(t)
		defer cache.Close()

		simulatedErr := errors.New("simulated data sync error")
		mockDataFile := newMockFile("mock.dat", make([]byte, 64))
		mockDataFile.syncErr = simulatedErr
		cache.data = mockDataFile // Replace real file with mock

		_, err := cache.WriteAt(make([]byte, 64), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError, got %T: %v", err, err)
		}
		if !errors.Is(err, simulatedErr) {
			t.Errorf("expected underlying error to be %v, but it was not found in chain", simulatedErr)
		}
	})

	t.Run("WriteAt index save error", func(t *testing.T) {
		cache, _, _ := setupTestCache(t)
		defer cache.Close()

		simulatedErr := errors.New("simulated index write error")
		mockIndexFile := newMockFile("mock.idx", nil)
		mockIndexFile.writeAtErr = simulatedErr
		cache.indexStore.wr = mockIndexFile // Replace real index file with mock

		_, err := cache.WriteAt(make([]byte, 64), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError, got %T: %v", err, err)
		}
		if !errors.Is(err, simulatedErr) {
			t.Errorf("expected underlying error to be %v, but it was not found in chain", simulatedErr)
		}
	})

	t.Run("WriteAt index short write", func(t *testing.T) {
		cache, _, _ := setupTestCache(t)
		defer cache.Close()

		mockIndexFile := newMockFile("mock.idx", nil)
		mockIndexFile.shortWrite = true
		cache.indexStore.wr = mockIndexFile // Replace real index file with mock

		_, err := cache.WriteAt(make([]byte, 64), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "failed to write all data to the index file") {
			t.Errorf("error message mismatch, got: %v", err)
		}
	})

	t.Run("ReadAt data file error", func(t *testing.T) {
		cache, _, _ := setupTestCache(t)
		defer cache.Close()

		// First, write data so the block is marked as cached
		if _, err := cache.WriteAt(make([]byte, 64), 0); err != nil {
			t.Fatalf("failed to write initial data: %v", err)
		}

		simulatedErr := errors.New("simulated data read error")
		mockDataFile := newMockFile("mock.dat", nil)
		mockDataFile.readAtErr = simulatedErr
		cache.data = mockDataFile // Replace real file with mock

		_, err := cache.ReadAt(make([]byte, 64), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError, got %T: %v", err, err)
		}
		if !errors.Is(err, simulatedErr) {
			t.Errorf("expected underlying error to be %v, but it was not found in chain", simulatedErr)
		}
	})

	t.Run("ReadAt short read", func(t *testing.T) {
		cache, _, _ := setupTestCache(t)
		defer cache.Close()

		// Write data so the block is marked as cached
		if _, err := cache.WriteAt(make([]byte, 64), 0); err != nil {
			t.Fatalf("failed to write initial data: %v", err)
		}

		// Replace real file with a mock that will perform a short read.
		// A short read from an io.ReaderAt returns io.EOF.
		mockDataFile := newMockFile("mock.dat", make([]byte, 63)) // 1 byte short
		cache.data = mockDataFile

		_, err := cache.ReadAt(make([]byte, 64), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError, got %T: %v", err, err)
		}
		if !errors.Is(err, io.EOF) {
			t.Errorf("expected underlying error to be io.EOF, but it was not found in chain")
		}
		if !strings.Contains(err.Error(), "failed to read all data") {
			t.Errorf("error message mismatch, got: %v", err)
		}
	})

	t.Run("uninitialized files", func(t *testing.T) {
		cache := &LocalDownloadCache{} // Create an uninitialized cache
		_, err := cache.WriteAt(make([]byte, 1), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError for WriteAt, got %v", err)
		}
		if !strings.Contains(err.Error(), "index store is not initialized") {
			t.Errorf("error message mismatch, got: %v", err)
		}

		_, err = cache.ReadAt(make([]byte, 1), 0)
		if !errors.Is(err, ErrCacheInternalError) {
			t.Errorf("expected ErrCacheInternalError for ReadAt, got %v", err)
		}
		if !strings.Contains(err.Error(), "index store is not initialized") {
			t.Errorf("error message mismatch, got: %v", err)
		}
	})
}

func TestInternalCacheErrorFormatting(t *testing.T) {
	underlyingErr := fmt.Errorf("specific cause")
	err := &internalCacheError{err: underlyingErr}

	if !errors.Is(err, ErrCacheInternalError) {
		t.Error("errors.Is should identify the error as ErrCacheInternalError")
	}

	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is should find the wrapped underlying error")
	}

	if unwrapped := errors.Unwrap(err); unwrapped != underlyingErr {
		t.Errorf("errors.Unwrap() returned %v, want %v", unwrapped, underlyingErr)
	}

	expectedMsg := "internal cache error: specific cause"
	if err.Error() != expectedMsg {
		t.Errorf("Error() returned %q, want %q", err.Error(), expectedMsg)
	}
}
