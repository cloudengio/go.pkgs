// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"io"
	"iter"
	"strconv"
	"time"

	"cloudeng.io/net/ratecontrol"
)

// RetryResponse allows the caller to determine whether an operation
// that failed with a retryable error can be retried and how long to wait
// before retrying the operation.
type RetryResponse interface {
	// IsRetryable checks if the error is retryable.
	IsRetryable() bool

	// BackoffDuration returns true if a specific backoff duration is specified
	// in the response, in which case the duration is returned. If false
	// no specific backoff duration is requested and the backoff algorithm
	// should fallback to something appropriate, such as exponential backoff.
	BackoffDuration() (bool, time.Duration)
}

type backoff struct {
	exponential ratecontrol.Backoff
	steps       int
	retries     int
}

// NewBackoff creates a new backoff instance that implements an
// exponential backoff algorithm unless the RetryResponse specifies
// a specific backoff duration. The backoff will continue for the
// specified number of steps, after which it will return true to indicate
// that no more retries should be attempted.
func NewBackoff(initial time.Duration, steps int) ratecontrol.Backoff {
	return &backoff{
		exponential: ratecontrol.NewExpontentialBackoff(initial, steps),
		steps:       steps,
		retries:     0,
	}
}

func (b *backoff) Retries() int {
	return b.retries
}

func (b *backoff) Wait(ctx context.Context, r any) (bool, error) {
	if b.retries >= b.steps {
		return true, nil
	}
	rr, ok := r.(RetryResponse)
	if !ok {
		return true, fmt.Errorf("expected RetryResponse, got %T", r)
	}
	ok, duration := rr.BackoffDuration()
	if !ok {
		return b.exponential.Wait(ctx, nil)
	}
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-time.After(duration):
	}
	b.retries++
	return false, nil
}

// Reader provides support for downloading very large files efficiently
// concurrently and to allow for resumption of partial downloads.
type Reader interface {
	Name() string // Name returns the name of the file being read.

	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize() (int64, int)

	// Digest returns the digest of the file, if available, the
	// format defined by RFC 9530's Repr-Digest header, eg.
	// Repr-Digest: sha-256=:d435Qo+nKZ+gLcUHn7GQtQ72hiBVAgqoLsZnZPiTGPk=:
	// An empty string is returned if no digest is available.
	Digest() string

	// GetReader retrieves a byte range from the file and returns
	// a reader that can be used to access that data range. In addition to the
	// error, the RetryResponse is returned which indicates whether the
	// operation can be retried and the duration to wait before retrying.
	GetReader(ctx context.Context, from, to int64) (io.ReadCloser, RetryResponse, error)
}

// ByteRange represents a range of bytes in a file.
// The range is inclusive of the 'From' byte and the 'To' byte as per
// the HTTP Range header specification/convention.
type ByteRange struct {
	From int64 // Inclusive start of the range.
	To   int64 // Inclusive end of the range.
}

func (br ByteRange) String() string {
	return "[" + strconv.FormatInt(br.From, 10) + ":" + strconv.FormatInt(br.To, 10) + "]"
}

func (br ByteRange) Size() int64 {
	// Size returns the size of the byte range.
	if br.From < 0 || br.To < br.From {
		return 0
	}
	return br.To - br.From + 1 // Inclusive range.
}

// NumBlocks calculates the number of blocks required to cover the content size
// given the specified block size. It returns the number of blocks needed.
// If the content size is not a multiple of the block size, it adds an additional
// block to cover the remaining bytes.
func NumBlocks(contentSize int64, blockSize int) int {
	nb := contentSize / int64(blockSize)
	if contentSize%int64(blockSize) != 0 {
		nb++
	}
	return int(nb)
}

// Ranges returns an iterator for all byte ranges over the specified range
// and block size. Each range is inclusive of the 'From' byte and the 'To' byte.
// The ranges are generated in blocks of the specified size, with the last block
// potentially being smaller than the specified block size.
func Ranges(from, to int64, blockSize int) iter.Seq[ByteRange] {
	size := to - from + 1
	return func(yield func(ByteRange) bool) {
		if blockSize <= 0 {
			return
		}
		nb := NumBlocks(size, blockSize)
		for i := range nb {
			start := from + int64(i*blockSize)
			end := min(start+int64(blockSize), to+1) - 1
			if !yield(ByteRange{From: start, To: end}) {
				return
			}
		}
	}
}

// RangeForIndex returns the byte range for the specified block index in
// a series of blocks of the specified size over the content size.
// The range is inclusive of the 'From' byte and the 'To' byte.
// If the index is out of bounds, it returns an invalid range with From and To set
// to -1.
func RangeForIndex(index int, contentSize int64, blockSize int) ByteRange {
	// rangeForIndex returns the byte range for the specified block index.
	if index < 0 || contentSize <= 0 || blockSize <= 0 {
		return ByteRange{From: -1, To: -1} // Invalid index or sizes.
	}
	from := int64(index * blockSize)
	to := min(from+int64(blockSize), contentSize) - 1
	return ByteRange{From: from, To: to}
}
