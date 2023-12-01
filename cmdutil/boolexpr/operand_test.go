// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr_test

import (
	"testing"

	"cloudeng.io/cmdutil/boolexpr"
)

func TestOperandRegistration(t *testing.T) {
	p := boolexpr.NewParser()
	p.RegisterOperand("newOp", func(n, v string) boolexpr.Operand { return regexOp{val: v} })
	m, err := p.Parse("newOp=foo")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m.Eval("foo"), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := m.Eval("bar"), false; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
