// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"cloudeng.io/cmdutil/boolexpr"
)

type commonOperand struct {
	requires reflect.Type
	document string
}

func (co commonOperand) Needs(t reflect.Type) bool {
	return t.Implements(co.requires)
}
func (co commonOperand) Document() string {
	return co.document
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

func (op regEx) Prepare() (boolexpr.Operand, error) {
	re, err := regexp.Compile(op.text)
	if err != nil {
		return op, err
	}
	op.re = re
	return op, nil
}

func (op regEx) Eval(v any) bool {
	if nt, ok := v.(nameIfc); ok {
		return op.re.MatchString(nt.Name())
	}
	return false
}

func (op regEx) String() string {
	return "re=" + op.text
}

// Regexp returns a regular expression operand. It is not compiled until
// a matcher.T is created using New. It requires that the value being
// matched provides Name() string.
func Regexp(re string) boolexpr.Operand {
	return regEx{text: re,
		commonOperand: commonOperand{
			document: "re=<regexp> matches a regular expression for any type that implements: Name() string",
			requires: reflect.TypeOf((*nameIfc)(nil)).Elem(),
		},
	}
}

type glob struct {
	text            string
	caseInsensitive bool
	commonOperand
}

func (op glob) Prepare() (boolexpr.Operand, error) {
	_, err := filepath.Match(op.text, "foo")
	if err != nil {
		return op, err
	}
	op.requires = reflect.TypeOf((*nameIfc)(nil)).Elem()
	return op, nil
}

func (op glob) Eval(v any) bool {
	if nt, ok := v.(nameIfc); ok {
		name := nt.Name()
		if op.caseInsensitive {
			name = strings.ToLower(name)
		}
		matched, _ := filepath.Match(op.text, name)
		return matched
	}
	return false
}

func (op glob) String() string {
	if op.caseInsensitive {
		return "iname=" + op.text
	}
	return "name=" + op.text
}

// Glob provides a glob operand that may be case insensitive, in which
// case the value it is being against will be converted to lower case
// before the match is evaluated. The pattern is not validated until a matcher.T
// is created using New.
func Glob(pat string, caseInsensitive bool) boolexpr.Operand {
	return glob{text: pat,
		caseInsensitive: caseInsensitive,
		commonOperand: commonOperand{
			document: "name=<glob> matches a glob pattern for any type that implements: Name() string",
			requires: reflect.TypeOf((*nameIfc)(nil)).Elem(),
		}}
}

type fileType struct {
	text string
	commonOperand
	// true if the operand requires a full mode, false if it only requires the modeType.
	needsMode bool
}

func (op fileType) Prepare() (boolexpr.Operand, error) {
	switch op.text {
	case "d", "l", "f":
		op.needsMode = false
		op.requires = reflect.TypeOf((*fileTypeIfc)(nil)).Elem()
	case "x":
		op.needsMode = true
		op.requires = reflect.TypeOf((*fileModeIfc)(nil)).Elem()
	default:
		return op, fmt.Errorf("invalid file type: %q, use one of d, f, l or x", op.text)
	}
	return op, nil
}

func (op fileType) Eval(v any) bool {
	var mode fs.FileMode
	switch t := v.(type) {
	case fileTypeIfc:
		mode = t.Type()
		if op.needsMode {
			// need the full fileMode, but only Type is available.
			return false
		}
	case fileModeIfc:
		mode = t.Mode()
	default:
		return false
	}
	switch op.text {
	case "f":
		return mode.IsRegular()
	case "x":
		return mode.IsRegular() && (mode.Perm()&0111 != 0)
	case "l":
		return mode&fs.ModeSymlink != 0
	case "d":
		return mode.IsDir()
	}
	return false
}

func (op fileType) String() string {
	return "type=" + op.text
}

func (op fileType) Needs(t reflect.Type) bool {
	return t.Implements(op.requires)
}

const fileTypeDoc = `"type=<type> matches a file type (d, f, t) for any type that implements: Type() fs.FileMode, and type 'x' if the type implements: Mode() fs.FileMode`

// FileType returns a 'file type' item. It is not validated until a
// matcher.T is created using New. Supported file types are
// (as per the unix find command):
//   - f for regular files
//   - d for directories
//   - l for symbolic links
//   - x executable regular files
//
// It requires that the value bein matched provides Mode() fs.FileMode or
// Type() fs.FileMode (which should return Mode&fs.ModeType).
func FileType(typ string) boolexpr.Operand {
	return fileType{
		text: typ,
		commonOperand: commonOperand{
			document: fileTypeDoc,
		}}
}

type newerThan struct {
	text string
	when time.Time
	commonOperand
}

func (op newerThan) Prepare() (boolexpr.Operand, error) {
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
	return "newer=" + op.text
}

const newerThanDoc = `"newer=<time> matches a time that is newer than the specified time for any type that implements: ModTime() time.Time. The time is specified in time.RFC3339, time.DateTime, time.TimeOnly or time.DateOnly formats",`

// NewerThanParsed returns a 'newer than' operand. It is not validated until a
// matcher.T is created using New. The time must be expressed as one of
// time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly. Due to the
// nature of the parsed formats fine grained time comparisons are not
// possible.
//
// It requires that the value bein matched provides ModTime() time.Time.
func NewerThanParsed(value string) boolexpr.Operand {
	return newerThan{text: value,
		commonOperand: commonOperand{
			document: newerThanDoc,
			requires: reflect.TypeOf((*modTimeIfc)(nil)).Elem(),
		}}
}

// NewerThanTime returns a 'newer than' operand with the specified time.
// This should be used in place of NewerThanFormat when fine grained time
// comparisons are required.
//
// It requires that the value bein matched provides ModTime() time.Time.
func NewerThanTime(when time.Time) boolexpr.Operand {
	return newerThan{when: when,
		commonOperand: commonOperand{
			document: newerThanDoc,
			requires: reflect.TypeOf((*modTimeIfc)(nil)).Elem(),
		}}
}

// NewGlob returns a case sensitive boolexpr.Operand that matches a glob pattern.
func NewGlob(_, v string) boolexpr.Operand { return Glob(v, false) }

// NewIGlob is a case-insensitive version of NewGlob.
func NewIGlob(_, v string) boolexpr.Operand { return Glob(v, true) }

// NewRegexp returns a boolexpr.Operand that matches a regular expression.
func NewRegexp(_, v string) boolexpr.Operand { return Regexp(v) }

// NewFileType returns a boolexpr.Operand that matches a file type.
func NewFileType(_, v string) boolexpr.Operand { return FileType(v) }

// NewNewerThan returns a boolexpr.Operand that matches a time that is newer
func NewNewerThan(_, v string) boolexpr.Operand { return NewerThanParsed(v) }

// New returns a boolexpr.Parser with the following operands registered:
//   - name=<glob> matches a glob pattern for any type that implements: Name() string
//   - iname=<glob> matches a case insensitive glob pattern for any type that implements: Name() string
//   - re=<regexp> matches a regular expression for any type that implements: Name() string
//   - type=<type> matches a file type (d, f, t) for any type that implements: Type() fs.FileMode, and type 'x' if the type implements: Mode() fs.FileMode
//   - newer=<time> matches a time that is newer than the specified time for any type that implements: ModTime() time.Time. The time is specified in time.RFC3339, time.DateTime, time.TimeOnly or time.DateOnly formats
func New() *boolexpr.Parser {
	parser := boolexpr.NewParser()
	parser.RegisterOperand("name", NewGlob)
	parser.RegisterOperand("iname", NewIGlob)
	parser.RegisterOperand("re", NewRegexp)
	parser.RegisterOperand("type", NewFileType)
	parser.RegisterOperand("newer", NewNewerThan)
	return parser
}
