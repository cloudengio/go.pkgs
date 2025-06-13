// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package errors_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"cloudeng.io/errors"
)

type ErrorStruct struct {
	X int
	S string
}

func (es ErrorStruct) Error() string {
	return es.S
}

func (es *ErrorStruct) Is(target error) bool {
	_, ok := target.(*ErrorStruct)
	return ok
}

func (es *ErrorStruct) As(target interface{}) bool {
	v, ok := target.(*ErrorStruct)
	if !ok {
		return false
	}
	*v = *es
	return ok
}

func TestMulti(t *testing.T) {
	assert := func(m *errors.M, e error, msg string) {
		if e == nil {
			if m.Err() != nil {
				t.Errorf("unexpected error: %v", m)
			}
			return
		}

		if got, want := m.Error(), e.Error(); got != want {
			t.Errorf("got %v, want %v", got, want)
		}

		if got, want := m.Error(), msg; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	m := &errors.M{}
	assert(m, nil, "")
	m.Append()
	assert(m, nil, "")
	m.Append(nil, nil)
	assert(m, nil, "")

	e1 := errors.New("x")
	m.Append(e1)
	assert(m, e1, "x")

	e2 := errors.New("y")

	m.Append(e2)
	assert(m, m, `  --- 1 of 2 errors
  x
  --- 2 of 2 errors
  y`)

	out := &strings.Builder{}
	m = &errors.M{}
	fmt.Fprintf(out, "%s\n", m)
	fmt.Fprintf(out, "%+v\n", m)
	fmt.Fprintf(out, "%#v\n", m)
	if got, want := out.String(), "\n\n\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	m = &errors.M{}
	out.Reset()
	ex := &ErrorStruct{S: "X"}
	m.Append(ex)
	fmt.Fprintf(out, "%v\n", m)
	fmt.Fprintf(out, "%+v\n", m)
	fmt.Fprintf(out, "%#v\n", m)
	if got, want := out.String(), `X
X
&errors_test.ErrorStruct{X:0, S:"X"}
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out.Reset()
	ey := &ErrorStruct{S: "Y"}
	m.Append(ey)
	fmt.Fprintf(out, "%v", m)
	fmt.Fprintf(out, "%+v", m)
	fmt.Fprintf(out, "%#v", m)
	if got, want := out.String(), `  --- 1 of 2 errors
  X
  --- 2 of 2 errors
  Y  --- 1 of 2 errors
  X
  --- 2 of 2 errors
  Y
  --- 1 of 2 errors
  &errors_test.ErrorStruct{X:0, S:"X"}
  --- 2 of 2 errors
  &errors_test.ErrorStruct{X:0, S:"Y"}
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUnWrap(t *testing.T) {
	m := &errors.M{}
	e1 := errors.New("1")
	e2 := errors.New("2")
	m.Append(e1, e2)
	if got, want := m.Unwrap(), []error{e1, e2}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Unwrap will not unwrap errors that support Unwrap() []error.
	if errors.Unwrap(m) != nil {
		t.Errorf("not empty")
	}
}

func TestClone(t *testing.T) {
	t1 := os.ErrExist
	t2 := &ErrorStruct{X: 2, S: "2"}
	t3 := &ErrorStruct{X: 2, S: "2"}
	m := &errors.M{}
	m.Append(t1, t2, t3)
	if got, want := m.Unwrap(), []error{t1, t2, t3}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	c := m.Clone()
	if got, want := c.Unwrap(), []error{t1, t2, t3}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	m.Append(os.ErrInvalid)
	if got, want := m.Unwrap(), []error{t1, t2, t3, os.ErrInvalid}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := c.Unwrap(), []error{t1, t2, t3}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAsIs(t *testing.T) {
	t1 := os.ErrExist
	t2 := &ErrorStruct{X: 2, S: "2"}
	m := &errors.M{}
	m.Append(t1, t2)
	if got, want := errors.Is(m, os.ErrExist), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errors.Is(m, t2), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errors.Is(m, &ErrorStruct{X: 22, S: "22"}), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errors.Is(m, os.ErrNoDeadline), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	t3 := &ErrorStruct{}
	if got, want := errors.As(m, t3), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := t3, t2; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	var rd io.Reader
	if got, want := errors.As(m, &rd), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestErr(t *testing.T) {
	m := &errors.M{}
	if m.Err() != nil {
		t.Errorf("unexpected non-nil error")
	}
	m.Append(os.ErrExist)
	err := m.Err()
	if err == nil {
		t.Errorf("expected an error")
	}
	if _, ok := err.(*errors.M); !ok {
		t.Errorf("failed to extract underlying type")
	}
}

func TestSquashNop(t *testing.T) {
	m := &errors.M{}
	m.Append(os.ErrExist)
	if got, want := m.Squash(os.ErrExist).Error(), `file already exists`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	err := m.Squash(os.ErrInvalid)
	if got, want := err.Error(), `file already exists`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := fmt.Sprintf("%v", err), `file already exists`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := errors.Is(err, os.ErrExist), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSquash(t *testing.T) {
	m := &errors.M{}
	m.Append(context.Canceled)
	m.Append(os.ErrExist)
	m.Append(os.ErrInvalid)
	m.Append(context.Canceled)

	msg := fmt.Sprintf("%v", m)
	if got, want := msg, `  --- 1 of 4 errors
  context canceled
  --- 2 of 4 errors
  file already exists
  --- 3 of 4 errors
  invalid argument
  --- 4 of 4 errors
  context canceled`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	msg = fmt.Sprintf("%v", m.Squash(context.Canceled))
	if got, want := msg, `  --- 1 of 3 errors
  context canceled (repeated 2 times)
  --- 2 of 3 errors
  file already exists
  --- 3 of 3 errors
  invalid argument`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	m = &errors.M{}
	m.Append(os.ErrInvalid)
	m.Append(context.Canceled)
	m.Append(context.Canceled)
	m.Append(os.ErrExist)
	m.Append(context.DeadlineExceeded)
	m.Append(context.Canceled)
	m.Append(context.Canceled)
	m.Append(context.DeadlineExceeded)

	if got, want := strings.Count(m.Error(), "context canceled"), 4; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	cancelSquash := m.Squash(context.Canceled)
	if got, want := cancelSquash.Error(), `  --- 1 of 5 errors
  invalid argument
  --- 2 of 5 errors
  context canceled (repeated 4 times)
  --- 3 of 5 errors
  file already exists
  --- 4 of 5 errors
  context deadline exceeded
  --- 5 of 5 errors
  context deadline exceeded`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := cancelSquash.(*errors.M).Squash(context.DeadlineExceeded).Error(), `  --- 1 of 4 errors
  invalid argument
  --- 2 of 4 errors
  context canceled (repeated 4 times)
  --- 3 of 4 errors
  file already exists
  --- 4 of 4 errors
  context deadline exceeded (repeated 2 times)`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	m = &errors.M{}
	m.Append(fmt.Errorf("e1 %w", context.Canceled))
	m.Append(fmt.Errorf("e2 %w", context.Canceled))
	m.Append(fmt.Errorf("e2 %w", context.Canceled))

	if got, want := m.Squash(context.Canceled).Error(), `e1 context canceled (repeated 3 times)`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

const recursiveError = `  --- 1 of 5 errors
  context canceled
  --- 2 of 5 errors
  file already exists
  --- 3 of 5 errors
  invalid argument
  --- 4 of 5 errors
  context canceled
  --- 5 of 5 errors
    --- 1 of 7 errors
    context canceled
    --- 2 of 7 errors
    invalid argument
    --- 3 of 7 errors
    context canceled
    --- 4 of 7 errors
      --- 1 of 4 errors
      context canceled
      --- 2 of 4 errors
      invalid argument
      --- 3 of 4 errors
      context canceled
      --- 4 of 4 errors
      context canceled
    --- 5 of 7 errors
    file already exists
    --- 6 of 7 errors
    context deadline exceeded
    --- 7 of 7 errors
    context deadline exceeded`

func TestSquashRecursive(t *testing.T) {
	m := &errors.M{}
	m.Append(context.Canceled)
	m.Append(os.ErrExist)
	m.Append(os.ErrInvalid)
	m.Append(context.Canceled)

	m2 := &errors.M{}
	m2.Append(context.Canceled)
	m2.Append(os.ErrInvalid)
	m2.Append(context.Canceled)
	m2.Append(context.Canceled)

	m1 := &errors.M{}
	m1.Append(context.Canceled)
	m1.Append(os.ErrInvalid)
	m1.Append(context.Canceled)
	m1.Append(m2)
	m1.Append(os.ErrExist)
	m1.Append(context.DeadlineExceeded)
	m1.Append(context.DeadlineExceeded)

	m.Append(m1)

	if got, want := m.Error(), recursiveError; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	out := fmt.Sprintf("%v", m)
	if got, want := out, recursiveError; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	m1s := m1.Squash(context.Canceled) //.(*errors.M).Squash(context.DeadlineExceeded)
	if got, want := m1s.Error(), `  --- 1 of 6 errors
  context canceled (repeated 2 times)
  --- 2 of 6 errors
  invalid argument
  --- 3 of 6 errors
    --- 1 of 2 errors
    context canceled (repeated 3 times)
    --- 2 of 2 errors
    invalid argument
  --- 4 of 6 errors
  file already exists
  --- 5 of 6 errors
  context deadline exceeded
  --- 6 of 6 errors
  context deadline exceeded`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	m1ss := m1s.(*errors.M).Squash(context.DeadlineExceeded)
	if got, want := m1ss.Error(), `  --- 1 of 5 errors
  context canceled (repeated 2 times)
  --- 2 of 5 errors
  invalid argument
  --- 3 of 5 errors
    --- 1 of 2 errors
    context canceled (repeated 3 times)
    --- 2 of 2 errors
    invalid argument
  --- 4 of 5 errors
  file already exists
  --- 5 of 5 errors
  context deadline exceeded (repeated 2 times)`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	squashed := errors.Squash(m, context.Canceled, context.DeadlineExceeded)
	if got, want := squashed.Error(), `  --- 1 of 4 errors
  context canceled (repeated 2 times)
  --- 2 of 4 errors
  file already exists
  --- 3 of 4 errors
  invalid argument
  --- 4 of 4 errors
    --- 1 of 5 errors
    context canceled (repeated 2 times)
    --- 2 of 5 errors
    invalid argument
    --- 3 of 5 errors
      --- 1 of 2 errors
      context canceled (repeated 3 times)
      --- 2 of 2 errors
      invalid argument
    --- 4 of 5 errors
    file already exists
    --- 5 of 5 errors
    context deadline exceeded (repeated 2 times)`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	squashed2 := m.Squash(context.Canceled, context.DeadlineExceeded)
	if got, want := squashed2.Error(), squashed.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

}

func TestAppend(t *testing.T) {

	n := &errors.M{}
	n.Append(os.ErrExist)
	n.Append(os.ErrInvalid)

	m := &errors.M{}
	m.Append(os.ErrExist)
	m.Append(os.ErrInvalid)

	if got, want := m.Unwrap(), []error{os.ErrExist, os.ErrInvalid}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	m.Append(n, os.ErrClosed)

	if got, want := m.Unwrap(), []error{os.ErrExist, os.ErrInvalid, n, os.ErrClosed}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	o := errors.NewM(os.ErrExist, os.ErrInvalid)
	p := errors.NewM(os.ErrClosed, o)
	m.Append(p)

	if got, want := m.Unwrap(), []error{os.ErrExist, os.ErrInvalid, n, os.ErrClosed, p}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func ExampleM() {
	m := &errors.M{}
	fmt.Println(m.Err())
	m.Append(os.ErrExist)
	m.Append(os.ErrInvalid)
	fmt.Println(m.Err())
	// Output:
	// <nil>
	//   --- 1 of 2 errors
	//   file already exists
	//   --- 2 of 2 errors
	//   invalid argument
}
