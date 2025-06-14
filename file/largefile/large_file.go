// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strconv"
	"sync"
	"time"

	"cloudeng.io/algo/container/bitmap"
	"cloudeng.io/net/ratecontrol"
)

// ChecksumType represents the type of checksum used for file integrity verification.
type ChecksumType int

const (
	NoChecksum ChecksumType = iota
	MD5
	SHA1
	CRC32C
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
	// ContentLengthAndBlockSize returns the total length of the file in bytes
	// and the preferred block size used for downloading the file.
	ContentLengthAndBlockSize(ctx context.Context) (int64, int, error)

	// Checksum returns the checksum type and the checksum value for the file,
	// if none are available then it returns NoChecksum and an empty string.
	Checksum(ctx context.Context) (ChecksumType, string, error)

	// GetReader retrieves a byte range from the file and returns
	// a reader that can be used to access that data range. In addition to the
	// error, the RetryResponse is returned which indicates whether the
	// operation can be retried and the duration to wait before retrying.
	GetReader(ctx context.Context, from, to int64) (io.ReadCloser, RetryResponse, error)
}

// ByteRange represents a range of bytes in a file.
// The range is inclusive of the 'From' byte and the 'To' byte as per
// the HTTP Range header specification.
type ByteRange struct {
	From int64 // Inclusive start of the range.
	To   int64 // Exclusive end of the range.
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

// ByteRange represents a collection of equally sized, contiguous, byte ranges
// that can be used to track which parts of a file to download or that have
// been downloaded.
type ByteRanges struct {
	mu          sync.RWMutex
	contentSize int64
	bitmapSize  int
	blockSize   int
	bitmap      bitmap.T
}

// NewByteRanges creates a new ByteRanges instance with the specified content size
// and block size. The content size is the total size of the file in bytes, and
// the block size is the size of each byte range in bytes.
func NewByteRanges(contentSize int64, blockSize int) *ByteRanges {
	nb := NumBlocks(contentSize, blockSize)
	return &ByteRanges{
		contentSize: contentSize,
		blockSize:   blockSize,
		bitmapSize:  nb,
		bitmap:      bitmap.New(nb),
	}
}

// NumBlocks returns the number of blocks required to cover the byte ranges
// represented by this ByteRanges instance.
func (br *ByteRanges) NumBlocks() int {
	// NumBlocks returns the number of blocks in the byte ranges.
	return br.bitmapSize
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

// MarshalJSON implements the json.Marshaler interface for ByteRanges.
func (br *ByteRanges) MarshalJSON() ([]byte, error) {
	jr, err := br.bitmap.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ranges := struct {
		ContentSize string          `json:"content_size"`
		BlockSize   int             `json:"block_size"`
		Ranges      json.RawMessage `json:"ranges"`
	}{
		ContentSize: strconv.FormatInt(br.contentSize, 10),
		BlockSize:   br.blockSize,
		Ranges:      jr,
	}
	return json.Marshal(ranges)
}

// UnmarshalJSON implements the json.Unmarshaler interface for ByteRanges.
func (br *ByteRanges) UnmarshalJSON(data []byte) error {
	var ranges struct {
		ContentSize string          `json:"content_size"`
		BlockSize   int             `json:"block_size"`
		Ranges      json.RawMessage `json:"ranges"`
	}
	if err := json.Unmarshal(data, &ranges); err != nil {
		return err
	}
	if ranges.ContentSize == "" || ranges.BlockSize <= 0 || len(ranges.Ranges) == 0 {
		return fmt.Errorf("invalid content size, block size, or empty ranges")
	}
	var err error
	br.contentSize, err = strconv.ParseInt(ranges.ContentSize, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid content size: %w", err)
	}
	br.blockSize = ranges.BlockSize
	br.bitmapSize = NumBlocks(br.contentSize, ranges.BlockSize)
	err = br.bitmap.UnmarshalJSON(ranges.Ranges)
	if err != nil {
		return fmt.Errorf("invalid bitmap data: %w", err)
	}
	return nil
}

func (br *ByteRanges) ContentLength() int64 {
	return br.contentSize
}

func (br *ByteRanges) BlockSize() int {
	return br.blockSize
}

func runIteratorAsync(it iter.Seq[int], contentSize int64, blockSize int, ch chan<- ByteRange) {
	go func() {
		defer close(ch)
		for i := range it {
			from := int64(i * blockSize)
			to := min(from+int64(blockSize), contentSize)
			ch <- ByteRange{From: from, To: to - 1}
		}
	}()
}

// NextClear returns an iterator for the next clear byte range starting from 'start'.
func (br *ByteRanges) NextClear(start int) iter.Seq[ByteRange] {
	ch := make(chan ByteRange)
	runIteratorAsync(br.NextClear(start), br.contentSize, br.blockSize, ch)

	/*
		return func(yield func(ByteRange) bool) {
			for i := range br.bitmap.NextClear(start, br.bitmapSize) {
				from := int64(i * br.blockSize)
				to := min(from+int64(br.blockSize), br.contentSize)
				if !yield(ByteRange{From: from, To: to - 1}) {
					return
				}
			}
		}
	*/
}

// NextSet returns an iterator for the next set byte range starting from 'start'.
func (br *ByteRanges) NextSet(start int) iter.Seq[ByteRange] {
	return func(yield func(ByteRange) bool) {
		for i := range br.bitmap.NextSet(start, br.bitmapSize) {
			from := int64(i * br.blockSize)
			to := min(from+int64(br.blockSize), br.contentSize)
			if !yield(ByteRange{From: from, To: to - 1}) {
				return
			}
		}
	}
}

// Set marks the byte range for the specified position as set.
// It has no effect if the position is out of bounds.
func (br *ByteRanges) Set(pos int64) {
	br.mu.Lock()
	defer br.mu.Unlock()
	// Set the byte range for the specified position.
	if pos < 0 || pos >= br.contentSize {
		return
	}
	blockIndex := int(pos) / br.blockSize
	if blockIndex < 0 || blockIndex >= br.bitmapSize {
		return
	}
	br.bitmap.Set(blockIndex)
}

// IsSet checks if the byte range for the specified position is set.
func (br *ByteRanges) IsSet(pos int64) bool {
	br.mu.RLock()
	defer br.mu.RUnlock()
	// Check if the position is out of bounds.
	if pos < 0 || pos >= br.contentSize {
		return false
	}
	blockIndex := int(pos) / br.blockSize
	if blockIndex < 0 || blockIndex >= br.bitmapSize {
		return false
	}
	return br.bitmap.IsSet(blockIndex)
}

// IsClear checks if the byte range for the specified position is clear.
func (br *ByteRanges) IsClear(pos int64) bool {
	br.mu.RLock()
	defer br.mu.RUnlock()
	// Check if the position is out of bounds.
	if pos < 0 || pos >= br.contentSize {
		return false
	}
	blockIndex := int(pos) / br.blockSize
	if blockIndex < 0 || blockIndex >= br.bitmapSize {
		return false
	}
	return !br.bitmap.IsSet(blockIndex)
}
