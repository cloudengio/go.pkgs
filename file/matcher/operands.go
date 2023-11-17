// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher

import (
	"fmt"
	"io/fs"
	"regexp"
	"time"
)

type regEx struct {
	text string
	re   *regexp.Regexp
}

func (op regEx) Prepare() (Operand, error) {
	re, err := regexp.Compile(op.text)
	if err != nil {
		return op, err
	}
	op.re = re
	return op, nil
}

func (op regEx) Eval(v Value) bool {
	return op.re.MatchString(v.Name())
}

func (op regEx) String() string {
	return op.text
}

// Regexp returns a regular expression operand. It is not compiled until
// a matcher.T is created using New.
func Regexp(re string) Item {
	return NewOperand(regEx{text: re})
}

type fileType struct {
	text string
	mode fs.FileMode
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
	return op, nil
}

func (op fileType) Eval(v Value) bool {
	mode := v.Mode()
	if op.text == "f" {
		return mode.IsRegular()
	}
	return mode&op.mode == op.mode
}

func (op fileType) String() string {
	return op.text
}

// FileType returns a 'file type' item. It is not validated until a
// matcher.T is created using New. Supported file types are
// (as per the unix find command):
//   - f for regular files
//   - d for directories
//   - l for symbolic links
func FileType(typ string) Item {
	return NewOperand(fileType{text: typ})
}

type newerThan struct {
	text string
	when time.Time
}

func (op newerThan) Prepare() (Operand, error) {
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		if t, err := time.Parse(format, op.text); err == nil {
			op.when = t
			return op, nil
		}
	}
	return op, fmt.Errorf("invalid time: %v, use one of RFC3339, Date and Time, Date or Time only formats", op.text)
}

func (op newerThan) Eval(v Value) bool {
	return v.ModTime().After(op.when)
}

func (op newerThan) String() string {
	return op.text
}

// NewerThan returns a 'newer than' item. It is not validated until a
// matcher.T is created using New.
func NewerThan(when string) Item {
	return NewOperand(newerThan{text: when})
}
