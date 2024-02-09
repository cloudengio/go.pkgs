// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"flag"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func TestMulti(t *testing.T) {
	ms := &flags.Repeating{}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Var(ms, "x", "repeating")
	if err := fs.Parse([]string{"-x=a", "-x=b"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, want := ms.String(), "a, b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	ms.Validate = func(_ string) error {
		return fmt.Errorf("oops")
	}
	err := fs.Parse([]string{"-x=a", "-x=b"})
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Fatalf("unexpected or missing error: %v", err)
	}

	if got, want := ms.Get().([]string), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCommaSeparated(t *testing.T) {
	ms := &flags.Commas{}
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Var(ms, "x", "commas")
	if err := fs.Parse([]string{"-x=a,x", "-x=b,y"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got, want := ms.String(), "a, x, b, y"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	ms.Validate = func(_ string) error {
		return fmt.Errorf("oops")
	}
	err := fs.Parse([]string{"-x=a", "-x=b"})
	if err == nil || !strings.Contains(err.Error(), "oops") {
		t.Fatalf("unexpected or missing error: %v", err)
	}
}
