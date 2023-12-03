// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package boolexpr

import (
	"reflect"
)

// Operand represents an operand. It is exposed to allow clients packages
// to define custom operands.
type Operand interface {
	// Prepare is used to prepare the operand for evaluation, for example, to
	// compile a regular expression. Document and String must be callable before
	// Prepare is called. Eval and Needs must only be called after Prepare.
	Prepare() (Operand, error)

	// Eval must return false for any type that it does not support.
	Eval(any) bool

	// Needs returns true if the operand needs the specified type.
	Needs(reflect.Type) bool

	// Document returns a string documenting the operand.
	Document() string

	// String returns a string representation of the operand and its current value.
	String() string
}

// NewOperandItem returns an item representing an operand.
func NewOperandItem(op Operand) Item {
	return Item{typ: operand, op: op}
}
