// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// annotated represents an error annotated with additional metadata
// such as the source code file and line number it was created from.
type annotated struct {
	annotation string
	err        error
}

// Error implements error.error
func (ae *annotated) Error() string {
	return fmt.Sprintf("%v: %v", ae.annotation, ae.err)
}

// Unwrap implements errors.Unwrap. It returns the stored error
// without the annotation.
func (ae *annotated) Unwrap() error {
	return ae.err
}

// Is supports errors.Is. It calls errors.Is with the stored error.
func (ae *annotated) Is(target error) bool {
	return errors.Is(ae.err, target)
}

// As supports errors.As. It calls errors.As with the stored error.
func (ae *annotated) As(target interface{}) bool {
	return errors.As(ae.err, target)
}

// Format implements fmt.Formatter.Format.
func (ae *annotated) Format(f fmt.State, c rune) {
	format := "%" + string(c)
	if !f.Flag('+') && !f.Flag('#') {
		fmt.Fprintf(f, format, ae.Error())
		return
	}
	switch {
	case f.Flag('+'):
		format = "%v: %+" + string(c)
	case f.Flag('#'):
		format = "%v: %#" + string(c)
	}
	fmt.Fprintf(f, format, ae.annotation, ae.err)
	return
}

// FileLocation returns the callers location as a filepath and line
// number. Depth follows the convention for runtime.Caller. The
// filepath is the trailing nameLen components of the filename
// returned by runtime.Caller. A nameLen of 2 is generally the
// best compromise between brevity and precision since it includes
// the enclosing directory component as well as the filename.
func FileLocation(depth, nameLen int) string {
	_, file, line, _ := runtime.Caller(depth)
	if nameLen <= 1 {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	base := ""
	for i := 0; i < nameLen; i++ {
		idx := strings.LastIndex(file, string(filepath.Separator))
		if idx < 0 {
			break
		}
		base = file[idx:] + base
		file = file[:idx]
	}
	if base[0] == '/' {
		base = base[1:]
	}
	return base + ":" + strconv.Itoa(line)
}

// Caller returns an error annotated with the location of its immediate caller.
func Caller(err error) error {
	return Annotate(FileLocation(2, 2), err)
}

// Annotate returns an error representing the original error and the
// supplied annotation.
func Annotate(annotation string, err error) error {
	return &annotated{
		annotation: annotation,
		err:        err,
	}
}

// AnnotateAll returns a slice of errors representing the original
// errors and the supplied annotation.
func AnnotateAll(annotation string, errs ...error) []error {
	result := make([]error, len(errs))
	for i, err := range errs {
		result[i] = Annotate(annotation, err)
	}
	return result
}
