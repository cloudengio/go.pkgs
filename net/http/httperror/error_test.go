// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httperror_test

import (
	"errors"
	"net/http"
	"testing"

	"cloudeng.io/net/http/httperror"
)

func TestError(t *testing.T) {
	// Make sure type for a nil error is error and not httperror.T.
	if got, want := httperror.CheckResponse(nil, &http.Response{StatusCode: 200}), (error)(nil); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
	if got, want := httperror.CheckResponseRetries(nil, &http.Response{StatusCode: 200}, 3), (error)(nil); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}

	abort := httperror.CheckResponse(http.ErrAbortHandler, nil)
	if !errors.Is(abort, http.ErrAbortHandler) {
		t.Errorf("got %#v, want %#v", abort, http.ErrAbortHandler)
	}

	bgway := httperror.CheckResponse(nil, &http.Response{StatusCode: http.StatusBadGateway})

	if !errors.Is(bgway, httperror.AsT(http.StatusBadGateway)) {
		t.Errorf("failed to match http status code")
	}

	if errors.Is(bgway, httperror.AsT(http.StatusAlreadyReported)) {
		t.Errorf("incorrectly matched http status code")
	}

	if !httperror.IsHTTPError(bgway, http.StatusBadGateway) {
		t.Errorf("failed to match http status code")
	}

	if httperror.IsHTTPError(bgway, http.StatusAlreadyReported) {
		t.Errorf("incorrectly matched http status code")
	}

}
