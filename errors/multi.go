// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// New calls errors.New.
func New(m string) error {
	return errors.New(m)
}

// Unwrap calls errors.Unwrap.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Is calls errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As calls errors.As.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// M represents multiple errors. It is thread safe. Typical usage is:
//
//	errs := errors.M{}
//	...
//	errs.Append(err)
//	...
//	return errs.Err()
type M struct {
	mu          sync.RWMutex
	errs        []error // GUARDED_BY(mu)
	numSquashed int
}

// NewM is equivalent to:
//
//	errs := errors.M{}
//	...
//	errs.Append(err)
//	...
//	return errs.Err()
func NewM(errs ...error) error {
	err := &M{}
	err.Append(errs...)
	return err.Err()
}

// Append appends the specified errors excluding nil values.
func (m *M) Append(errs ...error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, err := range errs {
		if err == nil {
			continue
		}
		if merr, ok := err.(*M); ok {
			m.errs = append(m.errs, merr.errs...)
			continue
		}
		m.errs = append(m.errs, err)
	}
}

// Unwrap implements errors.Unwrap() []error.
func (m *M) Unwrap() []error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errs
}

// Is supports errors.Is.
func (m *M) Is(target error) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, err := range m.errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// As supports errors.As.
func (m *M) As(target interface{}) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, err := range m.errs {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// Error implements error.error
func (m *M) Error() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l := len(m.errs)
	switch l {
	case 0:
		return ""
	case 1:
		if m.numSquashed <= 1 {
			return m.errs[0].Error()
		}
		return fmt.Sprintf("%v (repeated %d times)", m.errs[0], m.numSquashed)
	}
	out := &strings.Builder{}
	for i, err := range m.errs {
		fmt.Fprintf(out, "  --- %v of %v errors\n  ", i+1, l)
		out.WriteString(err.Error())
		out.WriteString("\n")
	}
	return strings.TrimSuffix(out.String(), "\n")
}

// Format implements fmt.Formatter.Format.
func (m *M) Format(f fmt.State, c rune) {
	format := "%" + string(c)
	if !f.Flag('+') && !f.Flag('#') {
		fmt.Fprintf(f, format, m.Error())
		return
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch {
	case f.Flag('+'):
		format = "%+" + string(c)
	case f.Flag('#'):
		format = "%#" + string(c)
	}
	l := len(m.errs)
	if l == 1 {
		fmt.Fprintf(f, format, m.errs[0])
		return
	}
	format += "\n"
	for i, err := range m.errs {
		fmt.Fprintf(f, "  --- %v of %v errors\n  ", i+1, l)
		fmt.Fprintf(f, format, err)
	}
}

func (m *M) squash(target error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.errs) == 0 {
		return nil
	}
	c := &M{errs: make([]error, 0, len(m.errs))}
	squashed := &M{errs: make([]error, 1)}
	for _, err := range m.errs {
		if errors.Is(err, target) {
			if squashed.numSquashed == 0 {
				squashed.errs[0] = err
				squashed.numSquashed = 1
				continue
			}
			squashed.numSquashed++
		} else {
			c.errs = append(c.errs, err)
		}
	}
	switch {
	case squashed.numSquashed == 1:
		c.errs = append(c.errs, squashed.errs[0])
	case squashed.numSquashed > 1:
		c.errs = append(c.errs, squashed)
	}
	return c
}

// Squash returns an error.M with at most one instance of target per
// level in the error tree.
func (m *M) Squash(target error) error {
	return m.squash(target)
}

// Err returns nil if m contains no errors, or itself otherwise.
func (m *M) Err() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.errs) == 0 {
		return nil
	}
	return m
}

// Clone returns a new errors.M that contains the same errors as itself.
func (m *M) Clone() *M {
	c := &M{}
	m.mu.RLock()
	defer m.mu.RUnlock()
	c.errs = make([]error, len(m.errs))
	copy(c.errs, m.errs)
	return c
}

// Squash returns an error that contains at most one instance of target
// per level in the error tree. If err is nil, it returns nil.
// If err is an *M, it calls Squash on that instance. Otherwise, it returns
// the original error.
func Squash(err error, target error) error {
	if err == nil {
		return nil
	}
	if m, ok := err.(*M); ok {
		return m.Squash(target)
	}
	return err
}
