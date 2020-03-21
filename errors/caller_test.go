// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package errors_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"cloudeng.io/errors"
)

func ExampleCaller() {
	err := errors.Caller(os.ErrNotExist)
	fmt.Printf("%v\n", err)
	fmt.Printf("%v\n", errors.Unwrap(err))
	// Output:
	// errors/caller_test.go:17: file does not exist
	// file does not exist
}

func ExampleM_caller() {
	m := &errors.M{}
	m.Append(errors.Caller(os.ErrExist))
	m.Append(errors.Caller(os.ErrInvalid))
	fmt.Println(m.Err())
	// Output:
	//   --- 1 of 2 errors
	//   errors/caller_test.go:27: file already exists
	//   --- 2 of 2 errors
	//   errors/caller_test.go:28: invalid argument
}

func ExampleFileLocation() {
	fmt.Println(errors.FileLocation(1, 1))
	fmt.Println(errors.FileLocation(1, 2))
	// Output:
	// caller_test.go:38
	// errors/caller_test.go:39
}

func TestAnnotated(t *testing.T) {
	myErr := &os.PathError{
		Op:   "open",
		Path: "/a/b",
		Err:  os.ErrNotExist,
	}
	err := errors.Caller(myErr)
	if got, want := err.Error(), "errors/caller_test.go:51: open /a/b: file does not exist"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errors.Unwrap(err), myErr; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errors.Is(err, myErr), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	var oErr *os.PathError
	if got, want := errors.As(err, &oErr), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := oErr, myErr; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	errs := errors.AnnotateAll("oh my", os.ErrClosed, os.ErrExist)
	if got, want := errs[0].Error(), "oh my: file already closed"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errs[1].Error(), "oh my: file already exists"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPrintf(t *testing.T) {
	err := errors.Caller(&os.PathError{
		Op:   "open",
		Path: "/a/b",
		Err:  os.ErrNotExist,
	})
	if got, want := fmt.Sprintf("%v", err), "errors/caller_test.go:78: open /a/b: file does not exist"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fmt.Sprintf("%+v", err), "errors/caller_test.go:78: open /a/b: file does not exist"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := fmt.Sprintf("%#v", err), `errors/caller_test.go:78: &os.PathError{Op:"open", Path:"/a/b", Err:(*errors.errorString)`; !strings.HasPrefix(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
