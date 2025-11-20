// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
	"slices"
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
func As(err error, target any) bool {
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
		m.errs = append(m.errs, err)
	}
}

// Unwrap implements errors.Unwrap() []error.
func (m *M) Unwrap() []error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.errs)
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
func (m *M) As(target any) bool {
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
	return m.error("  ")
}

func (m *M) error(indent string) string {
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
		out.WriteString(indent)
		fmt.Fprintf(out, "--- %v of %v errors\n", i+1, l)
		if me, ok := err.(*M); ok {
			if len(me.errs) < 2 {
				out.WriteString(indent)
			}
			out.WriteString(me.error(indent + "  "))
		} else {
			out.WriteString(indent)
			out.WriteString(err.Error())
		}
		out.WriteString("\n")
	}
	return strings.TrimSuffix(out.String(), "\n")
}

// Format implements fmt.Formatter.Format.
func (m *M) Format(f fmt.State, c rune) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.format(f, c, "  ")
}

func (m *M) format(f fmt.State, c rune, indent string) {
	format := "%" + string(c)
	if !f.Flag('+') && !f.Flag('#') {
		fmt.Fprintf(f, format, m.Error())
		return
	}
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
		fmt.Fprintf(f, "%v--- %v of %v errors\n%s", indent, i+1, l, indent)
		fmt.Fprintf(f, format, err)
	}
}

func (m *M) squashLevel(target error) *M {
	serr := make([]error, 0, len(m.errs))
	var squashed *M
	for _, child := range m.errs {
		_, ok := child.(*M)
		if !ok && errors.Is(child, target) {
			if squashed == nil {
				squashed = &M{errs: []error{child}, numSquashed: 1}
				serr = append(serr, squashed)
			} else {
				squashed.numSquashed++
			}
			continue
		}
		serr = append(serr, child)
	}
	n := &M{
		errs: slices.Clone(serr),
	}
	return n
}

func (m *M) squash(target error) *M {
	switch len(m.errs) {
	case 0:
		return nil
	case 1:
		return m
	}
	c := &M{errs: make([]error, 0, len(m.errs))}
	for _, child := range m.errs {
		if mChild, ok := child.(*M); ok {
			c.errs = append(c.errs, mChild.squash(target))
			continue
		}
		c.errs = append(c.errs, child)
	}
	c = c.squashLevel(target)
	return c
}

// Squash returns an error.M with at most one instance of each of the
// targets per level in the error tree.
func (m *M) Squash(targets ...error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s := m.Clone()
	for _, target := range targets {
		s = s.squash(target)
	}
	return s
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

// Squash returns an error that contains at most one instance of targets
// per level in the error tree. If err is nil, it returns nil.
// If err is an *M, it calls Squash on that instance. Otherwise, it returns
// the original error.
func Squash(err error, targets ...error) error {
	if err == nil {
		return nil
	}
	m, ok := err.(*M)
	if ok {
		return m.Squash(targets...)
	}
	return err
}
