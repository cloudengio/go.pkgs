// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"errors"
	"fmt"
)

var (
	ErrCacheInvalidBlockSize  = errors.New("invalid block size")
	ErrCacheInvalidOffset     = errors.New("invalid offset")
	ErrCacheUncachedRange     = errors.New("uncached range")
	ErrCacheInternalError     = &internalCacheError{}
	ErrStreamingInternalError = &internalStreamingError{}
	ErrDownloadInternalError  = &internalDownloadError{}
)

type internalCacheError struct {
	internalError
}

type internalStreamingError struct {
	internalError
}

type internalDownloadError struct {
	internalError
}

type internalError struct {
	err error
}

func (e *internalError) Error() string {
	return fmt.Sprintf("internal cache error: %v", e.err)
}

func (e *internalError) Unwrap() error {
	return e.err
}

func (e *internalCacheError) Is(target error) bool {
	_, ok := target.(*internalCacheError)
	return ok
}

func (e *internalStreamingError) Is(target error) bool {
	_, ok := target.(*internalStreamingError)
	return ok
}

func (e *internalDownloadError) Is(target error) bool {
	_, ok := target.(*internalDownloadError)
	return ok
}

func newInternalCacheError(err error) error {
	return &internalCacheError{internalError: internalError{err: err}}
}

func newInternalStreamingError(err error) error {
	return &internalStreamingError{internalError: internalError{err: err}}
}

func newInternalDownloadError(err error) error {
	return &internalDownloadError{internalError: internalError{err: err}}
}
