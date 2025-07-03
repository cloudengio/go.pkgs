// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"errors"
	"fmt"
)

var (
	ErrCacheInvalidBlockSize = errors.New("invalid block size")
	ErrCacheInvalidOffset    = errors.New("invalid offset")
	ErrCacheUncachedRange    = errors.New("uncached range")
	ErrInternalError         = &internalError{}
)

type internalError struct {
	component string
	err       error
}

func (e *internalError) Error() string {
	return fmt.Sprintf("%v: internal error: %v", e.component, e.err)
}

func (e *internalError) Unwrap() error {
	return e.err
}

func (e *internalError) Is(target error) bool {
	_, ok := target.(*internalError)
	return ok
}

func newInternalCacheError(err error) error {
	return &internalError{component: "cache", err: err}
}

func newInternalStreamingError(err error) error {
	return &internalError{component: "streaming", err: err}
}

func newInternalDownloadError(err error) error {
	return &internalError{component: "download", err: err}
}
