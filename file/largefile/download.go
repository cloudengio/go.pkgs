// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"io"
	"os"
)

const (
	partialSuffix = ".partial"
	indexSuffix   = ".index"
)

// Download downloads src to the file dest, resuming an earlier, incomplete,
// download of the same file if there is one. The download is made to
// dest+".partial", with its progress tracked in dest+".index", which are
// replaced by dest once the download is complete and, if src has a digest
// (see Reader.Digest), validated. If the download fails or is interrupted
// the partial files are retained and DownloadStatus.Resumable reports
// whether calling Download again will resume it. A previous partial download
// that is for a file of a different size or block size is discarded.
//
// The options accepted by NewCachingDownloader may be used, with the exception
// of WithDownloadDigest, which is derived from src.
func Download(ctx context.Context, src Reader, dest string, opts ...DownloadOption) (DownloadStatus, error) {
	partial, index := dest+partialSuffix, dest+indexSuffix
	var options downloadOptions
	for _, opt := range opts {
		opt(&options)
	}
	cache, err := openOrCreateCache(ctx, src, partial, index, options.concurrency)
	if err != nil {
		return DownloadStatus{}, err
	}
	dl, err := NewCachingDownloader(src, cache, opts...)
	if err != nil {
		cache.Close() //nolint:errcheck
		return DownloadStatus{}, err
	}
	status, err := dl.Run(ctx)
	if cerr := cache.Close(); err == nil {
		err = cerr
	}
	if err != nil || !status.Complete {
		return status, err
	}
	if err := validateDigest(src, partial); err != nil {
		// The data is corrupt, so there is nothing to resume.
		os.Remove(partial) //nolint:errcheck
		os.Remove(index)   //nolint:errcheck
		status.Complete, status.Resumable = false, false
		return status, err
	}
	if err := os.Remove(index); err != nil {
		return status, err
	}
	return status, os.Rename(partial, dest)
}

// openOrCreateCache opens the existing cache if it is for src, and creates a
// new one otherwise.
func openOrCreateCache(ctx context.Context, src Reader, partial, index string, concurrency int) (*LocalDownloadCache, error) {
	size, blockSize := src.ContentLengthAndBlockSize()
	dataExists, indexExists, err := CacheFilesExist(partial, index)
	if err != nil {
		return nil, err
	}
	if dataExists && indexExists {
		if cache, err := openCache(partial, index); err == nil {
			if csize, cblock := cache.ContentLengthAndBlockSize(); csize == size && cblock == blockSize {
				return cache, nil
			}
			cache.Close() //nolint:errcheck
		}
	}
	if err := CreateNewFilesForCache(ctx, partial, index, size, blockSize, concurrency, nil); err != nil {
		return nil, err
	}
	return openCache(partial, index)
}

func openCache(partial, index string) (*LocalDownloadCache, error) {
	data, idx, err := OpenCacheFiles(partial, index)
	if err != nil {
		return nil, err
	}
	cache, err := NewLocalDownloadCache(data, idx)
	if err != nil {
		data.Close() //nolint:errcheck
		idx.Close()  //nolint:errcheck
		return nil, err
	}
	return cache, nil
}

func validateDigest(src Reader, filename string) error {
	h := src.Digest()
	if !h.IsSet() {
		return nil
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	h.Reset()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if !h.Validate() {
		return fmt.Errorf("%v: %w", src.Name(), ErrDigestMismatch)
	}
	return nil
}
