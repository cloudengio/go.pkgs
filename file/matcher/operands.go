// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher

import (
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"time"
)

// Operand represents an operand. It is exposed to allow clients packages
// to define custom operands.
type Operand interface {
	// Prepare is used to prepare the operand for evaluation, for example, to
	// compile a regular expression.
	Prepare() (Operand, error)
	// Eval must return false for any type that it does not support.
	Eval(any) bool
	// Needs returns true if the operand needs the specified type.
	Needs(reflect.Type) bool
	String() string
}

// NewOperand returns an item representing an operand.
func NewOperand(op Operand) Item {
	return Item{typ: operand, op: op}
}

type commonOperand struct {
	requires reflect.Type
}

func (co commonOperand) Needs(t reflect.Type) bool {
	return t.Implements(co.requires)
}

type regEx struct {
	text string
	re   *regexp.Regexp
	commonOperand
}

type nameIfc interface {
	Name() string
}

type fileTypeIfc interface {
	Type() fs.FileMode
}

type fileModeIfc interface {
	Mode() fs.FileMode
}

type modTimeIfc interface {
	ModTime() time.Time
}

func (op regEx) Prepare() (Operand, error) {
	re, err := regexp.Compile(op.text)
	if err != nil {
		return op, err
	}
	op.re = re
	op.requires = reflect.TypeOf((*nameIfc)(nil)).Elem()
	return op, nil
}

func (op regEx) Eval(v any) bool {
	if nt, ok := v.(nameIfc); ok {
		return op.re.MatchString(nt.Name())
	}
	return false
}

func (op regEx) String() string {
	return op.text
}

// Regexp returns a regular expression operand. It is not compiled until
// a matcher.T is created using New. It requires that the value being
// matched provides Name() string.
func Regexp(re string) Item {
	return NewOperand(regEx{text: re})
}

type fileType struct {
	text string
	mode fs.FileMode
	commonOperand
	typeRequires reflect.Type
}

func (op fileType) Prepare() (Operand, error) {
	switch op.text {
	case "d":
		op.mode = fs.ModeDir
	case "f":
		op.mode = 0
	case "l":
		op.mode = fs.ModeSymlink
	default:
		return op, fmt.Errorf("invalid file type: %v, use one of d, f or l", op.text)
	}
	op.requires = reflect.TypeOf((*fileModeIfc)(nil)).Elem()
	op.typeRequires = reflect.TypeOf((*fileTypeIfc)(nil)).Elem()
	return op, nil
}

func (op fileType) Eval(v any) bool {
	var mode fs.FileMode
	switch t := v.(type) {
	case fileTypeIfc:
		mode = t.Type()
	case fileModeIfc:
		mode = t.Mode()
	default:
		return false
	}
	if op.text == "f" {
		return mode.IsRegular()
	}
	return mode&op.mode == op.mode
}

func (op fileType) String() string {
	return op.text
}

func (op fileType) Needs(t reflect.Type) bool {
	return t.Implements(op.requires) || t.Implements(op.typeRequires)
}

// FileType returns a 'file type' item. It is not validated until a
// matcher.T is created using New. Supported file types are
// (as per the unix find command):
//   - f for regular files
//   - d for directories
//   - l for symbolic links
//
// It requires that the value bein matched provides Mode() fs.FileMode or
// Type() fs.FileMode (which should return Mode&fs.ModeType).
func FileType(typ string) Item {
	return NewOperand(fileType{text: typ})
}

type newerThan struct {
	text string
	when time.Time
	commonOperand
}

func (op newerThan) Prepare() (Operand, error) {
	if !op.when.IsZero() {
		op.requires = reflect.TypeOf((*modTimeIfc)(nil)).Elem()
		return op, nil
	}
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		if t, err := time.Parse(format, op.text); err == nil {
			op.when = t
			op.requires = reflect.TypeOf((*modTimeIfc)(nil)).Elem()
			return op, nil
		}
	}
	return op, fmt.Errorf("invalid time: %v, use one of RFC3339, Date and Time, Date or Time only formats", op.text)
}

func (op newerThan) Eval(v any) bool {
	if nt, ok := v.(modTimeIfc); ok {
		return nt.ModTime().After(op.when)
	}
	return false
}

func (op newerThan) String() string {
	return op.text
}

// NewerThanParsed returns a 'newer than' operand. It is not validated until a
// matcher.T is created using New. The time must be expressed as one of
// time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly. Due to the
// nature of the parsed formats fine grained time comparisons are not
// possible.
//
// It requires that the value bein matched provides ModTime() time.Time.
func NewerThanParsed(when string) Item {
	return NewOperand(newerThan{text: when})
}

// NewerThanTime returns a 'newer than' operand with the specified time.
// This should be used in place of NewerThanFormat when fine grained time
// comparisons are required.
//
// It requires that the value bein matched provides ModTime() time.Time.
func NewerThanTime(when time.Time) Item {
	return NewOperand(newerThan{when: when})
}
