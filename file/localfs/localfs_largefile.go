// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localfs

import (
	"context"
	"io"
	"io/fs"
	"os"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file/largefile"
)

// LargeFile is a wrapper around a file that supports reading large files in
// blocks. It implements the largefile.Reader interface.
type LargeFile struct {
	f         *os.File
	blockSize int
	size      int64
	digest    digests.Hash
}

const DefaultLargeFileBlockSize = 1024 * 1024 * 16 // Default block size is 16 MiB.

// NewLargeFile creates a new LargeFile instance that wraps the provided file
// and uses the specified block size for reading. If the file does not exist or
// cannot be opened, an error is returned. The supplied digest is simply
// returned by the Digest() method and is not used to validate the file's
// contents directly.
func NewLargeFile(file *os.File, blockSize int, digest digests.Hash) (*LargeFile, error) {
	if blockSize <= 0 {
		blockSize = DefaultLargeFileBlockSize
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	if info.IsDir() {
		file.Close()
		return nil, fs.ErrInvalid
	}
	size := info.Size()
	return &LargeFile{
		f:         file,
		blockSize: blockSize,
		size:      size,
		digest:    digest,
	}, nil
}

type noRetry struct{}

func (noRetry) IsRetryable() bool {
	return false
}

func (noRetry) BackoffDuration() (bool, time.Duration) {
	return false, 0
}

// Name implements largefile.Reader.
func (lf *LargeFile) Name() string {
	return lf.f.Name()
}

// ContentLengthAndBlockSize implements largefile.Reader.
func (lf *LargeFile) ContentLengthAndBlockSize() (int64, int) {
	return lf.size, lf.blockSize
}

// Digest implements largefile.Reader.
func (lf *LargeFile) Digest() digests.Hash {
	return lf.digest
}

// GetReader implements largefile.Reader.
func (lf *LargeFile) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	return reader{f: lf.f, at: from}, noRetry{}, nil
}

type reader struct {
	f  *os.File
	at int64
}

func (r reader) Read(p []byte) (int, error) {
	return r.f.ReadAt(p, r.at)
}

func (r reader) Close() error {
	return nil
}
