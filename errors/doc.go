// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package errors provides utility routines for working with errors that
// are compatible with go 1.13+ and for annotating errors. It provides
// errors.M which can be used to collect and work with multiple errors in a
// thread safe manner. It also provides convenience routines for annotating
// existing errors with caller and other information.
//
//	errs := errors.M{}
//	errs.Append(fn(a))
//	errs.Append(fn(b))
//	err := errs.Err()
//
// The location of a function's immediate caller (depth of 1) in form of the
// directory/filename:<line> (name len of 2) can be obtained as follows:
//
//	errors.Caller(1, 2)
//
// Annotations, can be added as follows:
//
//	err := errors.WithCaller(os.ErrNotExist)
//
// Where:
//
//	fmt.Printf("%v\n", err)
//	fmt.Printf("%v\n", errors.Unwrap(err))
//
// Would produce:
//
//	errors/caller_test.go:17: file does not exist
//	file does not exist
//
// Annotated errors can be passed to errors.M:
//
//	errs := errors.M{}
//	errs.Append(errors.WithCaller(fn(a)))
//	errs.Append(errors.WithCaller(fn(b)))
//	err := errs.Err()
package errors //nolint:revive // var-naming: avoid package names that conflict with Go standard library package names (revive)
