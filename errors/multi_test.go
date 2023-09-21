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
	if got, want := errors.Unwrap(m), e1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := errors.Unwrap(m).Error(), e2.Error(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if errors.Unwrap(m) != nil {
		t.Errorf("not empty")
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

func TestClone(t *testing.T) {
	t1 := os.ErrExist
	t2 := &ErrorStruct{X: 2, S: "2"}
	m := &errors.M{}
	m.Append(t1, t2)
	c := m.Clone()
	if err := m.Unwrap(); err == nil {
		t.Fatal("error expected")
	}
	if got, want := m.Unwrap(), t2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := c.Unwrap(), t1; got != want {
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

func TestRemoveCancel(t *testing.T) {
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
	msg = fmt.Sprintf("%v", m.WithoutContextCanceled())
	if got, want := msg, `  --- 1 of 2 errors
  file already exists
  --- 2 of 2 errors
  invalid argument`; got != want {
		t.Errorf("got %v, want %v", got, want)
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
