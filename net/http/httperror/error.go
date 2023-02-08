// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httperror

import (
	"errors"
	"fmt"
	"net/http"
)

// T represents an error encountered while making an HTTP request of some
// form. The error may be the result of a failed local operation (in which
// case Err will be non-nil), or an error returned by the remote server (in
// which case Err will be nil but StatusCode will something other than
// http.StatusOK). In all cases, Err or StatusCode must contain an error.
type T struct {
	Err        error
	Status     string
	StatusCode int
	Retries    int
}

// Error implements error.
func (err *T) Error() string {
	if err.Err == nil {
		if len(err.Status) > 0 {
			return err.Status
		}
		return fmt.Sprintf("%v", err.StatusCode)
	}
	return err.Err.Error()
}

func (err *T) is(target error) bool {
	if err.Err != nil {
		return errors.Is(err.Err, target)
	}
	return false
}

// Is implements errors.Is.
func (err *T) Is(target error) bool {
	terr, ok := target.(*T)
	if !ok {
		return err.is(target)
	}
	if err.Err != nil {
		return err.is(terr.Err)
	}
	return err.StatusCode == terr.StatusCode
}

// CheckResponse creates a new instance of T given an error and http.Response
// returned by an http request operation (e.g. Get, Do etc).
// If err is nil, resp must not be nil. It will return nil if
// err is nil and resp.StatusCode is http.StatusOK. Otherwise, it will
// create an instance of httperror.T with the appropriate fields set.
func CheckResponse(err error, resp *http.Response) error {
	if err == nil && resp.StatusCode == http.StatusOK {
		return nil // ensure return type is error
	}
	if err != nil {
		return &T{Err: err}
	}
	return &T{
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
	}
}

// CheckResponseRetries is like Checkresponse but will set the retries
// field.
func CheckResponseRetries(err error, resp *http.Response, retries int) error {
	if err := CheckResponse(err, resp); err != nil {
		terr := err.(*T)
		terr.Retries = retries
		return terr
	}
	return nil
}

// AsT returns an httperror.T for the specified http status code.
func AsT(httpStatusCode int) *T {
	return &T{StatusCode: httpStatusCode}
}

// IsHTTPError returns true if err contains the specified http status code.
func IsHTTPError(err error, httpStatusCode int) bool {
	terr, ok := err.(*T)
	if !ok || terr.Err != nil {
		return false
	}
	return terr.StatusCode == httpStatusCode
}
