// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"errors"
	"testing"
)

func TestInternalErrorTypes(t *testing.T) {
	cacheErr := &internalCacheError{err: errors.New("cache")}
	streamErr := &internalStreamingError{err: errors.New("stream")}
	downloadErr := &internalDownloadError{err: errors.New("download")}

	// Test Is for internalCacheError
	if !errors.Is(cacheErr, &internalCacheError{}) {
		t.Errorf("internalCacheError should be recognized by errors.Is")
	}
	if errors.Is(cacheErr, &internalStreamingError{}) {
		t.Errorf("internalCacheError should not be recognized as internalStreamingError")
	}
	if errors.Is(cacheErr, &internalDownloadError{}) {
		t.Errorf("internalCacheError should not be recognized as internalDownloadError")
	}

	// Test Is for internalStreamingError
	if !errors.Is(streamErr, &internalStreamingError{}) {
		t.Errorf("internalStreamingError should be recognized by errors.Is")
	}
	if errors.Is(streamErr, &internalCacheError{}) {
		t.Errorf("internalStreamingError should not be recognized as internalCacheError")
	}
	if errors.Is(streamErr, &internalDownloadError{}) {
		t.Errorf("internalStreamingError should not be recognized as internalDownloadError")
	}

	// Test Is for internalDownloadError
	if !errors.Is(downloadErr, &internalDownloadError{}) {
		t.Errorf("internalDownloadError should be recognized by errors.Is")
	}
	if errors.Is(downloadErr, &internalCacheError{}) {
		t.Errorf("internalDownloadError should not be recognized as internalCacheError")
	}
	if errors.Is(downloadErr, &internalStreamingError{}) {
		t.Errorf("internalDownloadError should not be recognized as internalStreamingError")
	}
}
