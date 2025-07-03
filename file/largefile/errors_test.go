// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"fmt"
	"testing"

	"cloudeng.io/errors"
)

func TestErrorTypes(t *testing.T) {
	if !errors.Is(fmt.Errorf("... %w", ErrCacheInvalidBlockSize), ErrCacheInvalidBlockSize) {
		t.Errorf("ErrCacheInvalidBlockSize should be equal to itself")
	}
	if !errors.Is(fmt.Errorf("... %w", ErrCacheInvalidOffset), ErrCacheInvalidOffset) {
		t.Errorf("ErrCacheInvalidOffset should be equal to itself")
	}
	if !errors.Is(fmt.Errorf("... %w", ErrCacheUncachedRange), ErrCacheUncachedRange) {
		t.Errorf("ErrCacheUncachedRange should be equal to itself")
	}
}

func TestInternalError(t *testing.T) {
	cacheErr := newInternalCacheError(errors.New("test error"))
	downloadErr := newInternalDownloadError(errors.New("test error"))
	streamingErr := newInternalStreamingError(errors.New("test error"))

	if !errors.Is(cacheErr, ErrInternalError) {
		t.Errorf("cacheErr should be an internal error")
	}
	if !errors.Is(downloadErr, ErrInternalError) {
		t.Errorf("downloadErr should be an internal error")
	}
	if !errors.Is(streamingErr, ErrInternalError) {
		t.Errorf("streamingErr should be an internal error")
	}

	if got, want := cacheErr.Error(), "cache: internal error: test error"; got != want {
		t.Errorf("cacheErr.Error() = %q, want %q", got, want)
	}
	if got, want := downloadErr.Error(), "download: internal error: test error"; got != want {
		t.Errorf("downloadErr.Error() = %q, want %q", got, want)
	}
	if got, want := streamingErr.Error(), "streaming: internal error: test error"; got != want {
		t.Errorf("streamingErr.Error() = %q, want %q", got, want)
	}
}
