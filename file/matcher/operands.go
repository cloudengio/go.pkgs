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
	"strconv"
	"strings"
	"time"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
	"cloudeng.io/file/diskusage"
)

type commonOperand struct {
	requires reflect.Type
	name     string
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

// NameIfc and/or PathIfc must be implemented by any values that are used with
// the Glob operands.
type NameIfc interface {
	Name() string
}

// PathIfc must be implemented by any values that are used with the
// Regexp operand optionally for the Glob operand.
type PathIfc interface {
	Path() string
}

// FileTypeIfc must be implemented by any values that are used with the
// Filetype operand for types f, d or l.
type FileTypeIfc interface {
	Type() fs.FileMode
}

// FileModeIfc must be implemented by any values that are used with the
// Filetype operand for type x.
type FileModeIfc interface {
	Mode() fs.FileMode
}

// ModTimeIfc must be implemented by any values that are used with the
// NewerThan operand.
type ModTimeIfc interface {
	ModTime() time.Time
}

// FileSizeIfc must be implemented by any values that are used with the
// FileSize operand.
type FileSizeIfc interface {
	Size() int64
}

// DirSizeIfc must be implemented by any values that are used with the
// DirSize operand.
type DirSizeIfc interface {
	NumEntries() int64
}

// XAttrIfc must be implemented by any values that are used with the
// XAttr operand.
type XAttrIfc interface {
	XAttr() file.XAttr
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
	if nt, ok := v.(PathIfc); ok {
		return op.re.MatchString(nt.Path())
	}
	return false
}

func (op regEx) String() string {
	return "re=" + op.text
}

// Regexp returns a regular expression operand. It is not compiled until
// a matcher.T is created using New. It requires that the value being
// matched implements PathIfc.
func Regexp(opname string, re string) boolexpr.Operand {
	return regEx{text: re,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + "=<regexp> matches a regular expression",
			requires: reflect.TypeOf((*PathIfc)(nil)).Elem(),
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
	op.requires = reflect.TypeOf((*NameIfc)(nil)).Elem()
	return op, nil
}

func (op glob) eval(v string) bool {
	if op.caseInsensitive {
		v = strings.ToLower(v)
	}
	matched, _ := filepath.Match(op.text, v)
	return matched
}

func (op glob) Eval(v any) bool {
	// try name first, then path.
	if nt, ok := v.(NameIfc); ok {
		if op.eval(nt.Name()) {
			return true
		}
	}
	if pt, ok := v.(PathIfc); ok {
		if op.eval(pt.Path()) {
			return true
		}
	}
	return false
}

func (op glob) String() string {
	return op.name + "=" + op.text
}

// Glob provides a glob operand (optionally case insensitive, in which
// case the value it is being against will be converted to lower case
// before the match is evaluated). The pattern is not validated until a
// matcher.T is created. It requires that the value being matched implements
// NameIfc and/or PathIfc.
// The NameIfc interface is used first, if the value does not implement
// NameIfc or the glob evaluates to false, then PathIfc is used.
func Glob(opname string, pat string, caseInsensitive bool) boolexpr.Operand {
	return glob{text: pat,
		caseInsensitive: caseInsensitive,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + "=<glob> matches a glob pattern",
			requires: reflect.TypeOf((*NameIfc)(nil)).Elem(),
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
		op.requires = reflect.TypeOf((*FileTypeIfc)(nil)).Elem()
	case "x":
		op.needsMode = true
		op.requires = reflect.TypeOf((*FileModeIfc)(nil)).Elem()
	default:
		return op, fmt.Errorf("invalid file type: %q, use one of d, f, l or x", op.text)
	}
	return op, nil
}

func (op fileType) Eval(v any) bool {
	var mode fs.FileMode
	switch t := v.(type) {
	case FileTypeIfc:
		mode = t.Type()
		if op.needsMode {
			// need the full fileMode, but only Type is available.
			return false
		}
	case FileModeIfc:
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
	return op.name + "=" + op.text
}

func (op fileType) Needs(t reflect.Type) bool {
	return t.Implements(op.requires)
}

const fileTypeDoc = "=<type> matches a file type (d, f, l, x), where d is a directory, f a regular file, l a symbolic link and x an executable regular file"

// FileType returns a 'file type' operand. It is not validated until a
// matcher.T is created using New. Supported file types are
// (as per the unix find command):
//   - f for regular files
//   - d for directories
//   - l for symbolic links
//   - x executable regular files
//
// It requires that the value being matched implements FileTypeIfc for types
// d, f and l and FileModeIfc for type x.
func FileType(opname string, typ string) boolexpr.Operand {
	return fileType{
		text: typ,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + fileTypeDoc,
		}}
}

type newerThan struct {
	text string
	when time.Time
	commonOperand
}

func (op newerThan) Prepare() (boolexpr.Operand, error) {
	if !op.when.IsZero() {
		op.requires = reflect.TypeOf((*ModTimeIfc)(nil)).Elem()
		return op, nil
	}
	for _, format := range []string{time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly} {
		if t, err := time.Parse(format, op.text); err == nil {
			op.when = t
			op.requires = reflect.TypeOf((*ModTimeIfc)(nil)).Elem()
			return op, nil
		}
	}
	return op, fmt.Errorf("invalid time: %v, use one of RFC3339, Date and Time, Date or Time only formats", op.text)
}

func (op newerThan) Eval(v any) bool {
	if nt, ok := v.(ModTimeIfc); ok {
		return nt.ModTime().After(op.when)
	}
	return false
}

func (op newerThan) String() string {
	return op.name + "=" + op.text
}

const newerThanDoc = "=<time> matches a time that is newer than the specified time in time.RFC3339, time.DateTime, time.TimeOnly or time.DateOnly formats"

// NewerThanParsed returns a 'newer than' operand. It is not validated until a
// matcher.T is created using New. The time must be expressed as one of
// time.RFC3339, time.DateTime, time.TimeOnly, time.DateOnly. Due to the
// nature of the parsed formats fine grained time comparisons are not
// possible.
//
// It requires that the value being matched implements ModTimeIfc.
func NewerThanParsed(opname string, value string) boolexpr.Operand {
	return newerThan{text: value,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + newerThanDoc,
			requires: reflect.TypeOf((*ModTimeIfc)(nil)).Elem(),
		}}
}

// NewerThanTime returns a 'newer than' operand with the specified time.
// This should be used in place of NewerThanFormat when fine grained time
// comparisons are required.
//
// It requires that the value bein matched implements Mod
func NewerThanTime(opname string, when time.Time) boolexpr.Operand {
	return newerThan{when: when,
		commonOperand: commonOperand{
			name:     opname,
			document: opname + newerThanDoc,
			requires: reflect.TypeOf((*ModTimeIfc)(nil)).Elem(),
		}}
}

// DirSize returns a 'directory size' operand. The value is not validated
// until a matcher.T is created using New. The size must be expressed as
// an integer. If larger is true then the comparison is performed using
// >, otherwise <=.
// The operand requires that the value being matched implements DirSizeIfc.
func DirSize(opname, value string, larger bool) boolexpr.Operand {
	return dirSize{
		sizeCommon: sizeCommon{
			text:   value,
			larger: larger,
		},
		commonOperand: commonOperand{
			name:     opname,
			document: opname + "=<size> matches a directory size",
			requires: reflect.TypeOf((*DirSizeIfc)(nil)).Elem(),
		}}
}

func DirSizeLarger(n, v string) boolexpr.Operand {
	return DirSize(n, v, true)
}

func DirSizeSmaller(n, v string) boolexpr.Operand {
	return DirSize(n, v, false)
}

type sizeCommon struct {
	text   string
	larger bool
	a      int64
}

type dirSize struct {
	sizeCommon
	commonOperand
}

func (op dirSize) Prepare() (boolexpr.Operand, error) {
	s, err := strconv.ParseInt(op.text, 10, 64)
	if err != nil {
		return op, err
	}
	op.a = s
	return op, nil
}

func (op dirSize) Eval(v any) bool {
	if nt, ok := v.(DirSizeIfc); ok {
		if op.larger {
			return nt.NumEntries() > op.a
		}
		return nt.NumEntries() <= op.a
	}
	return false
}

func (op dirSize) String() string {
	return op.name + "=" + op.text
}

// FileSize returns a 'file size' operand. The value is not validated
// until a matcher.T is created using New. The size may be expressed as
// an in binary (GiB, KiB) or decimal (GB, KB) or as bytes
// (eg. 1.1GB, 1GiB or 1000). If larger is true then the comparison is performed
// using >, otherwise <=.
// The operand requires that the value being matched implements FileSizeIfc.
func FileSize(opname, value string, larger bool) boolexpr.Operand {
	return fileSize{
		sizeCommon: sizeCommon{
			text:   value,
			larger: larger,
		},
		commonOperand: commonOperand{
			name:     opname,
			document: opname + "=<size> matches a directory size",
			requires: reflect.TypeOf((*FileSizeIfc)(nil)).Elem(),
		}}
}

func FileSizeLarger(n, v string) boolexpr.Operand {
	return FileSize(n, v, true)
}

func FileSizeSmaller(n, v string) boolexpr.Operand {
	return FileSize(n, v, false)
}

type fileSize struct {
	sizeCommon
	commonOperand
}

func (op fileSize) Prepare() (boolexpr.Operand, error) {
	s, err := diskusage.ParseToBytes(op.text)
	if err != nil {
		return op, err
	}
	op.a = int64(s)
	return op, nil
}

func (op fileSize) Eval(v any) bool {
	if nt, ok := v.(FileSizeIfc); ok {
		if op.larger {
			return nt.Size() > op.a
		}
		return nt.Size() <= op.a
	}
	return false
}

func (op fileSize) String() string {
	return op.name + "=" + op.text
}

// NewGlob returns a case sensitive boolexpr.Operand that matches a glob pattern.
// The expression value must implement NameIfc.
func NewGlob(n, v string) boolexpr.Operand { return Glob(n, v, false) }

// NewIGlob is a case-insensitive version of NewGlob.
// The expression value must implement NameIfc.
func NewIGlob(n, v string) boolexpr.Operand { return Glob(n, v, true) }

// NewRegexp returns a boolexpr.Operand that matches a regular expression.
// The expression value must implement NameIfc.
func NewRegexp(n, v string) boolexpr.Operand { return Regexp(n, v) }

// NewFileType returns a boolexpr.Operand that matches a file type.
// The expression value must implement FileTypeIfc for types d, f and l and
// FileModeIfc for type x.
func NewFileType(n, v string) boolexpr.Operand { return FileType(n, v) }

// NewNewerThan returns a boolexpr.Operand that matches a time that is newer
// than the specified time. The time is specified in time.RFC3339, time.DateTime,
// time.TimeOnly or time.DateOnly formats. The expression value must implement
// ModTimeIfc.
func NewNewerThan(n, v string) boolexpr.Operand { return NewerThanParsed(n, v) }

// NewDirSizeLarger returns a boolexpr.Operand that returns true if the expression
// value implements DirSizeIfc and the number of entries in the directory
// is greater than the specified value.
func NewDirSizeLarger(n, v string) boolexpr.Operand { return DirSizeLarger(n, v) }

// NewDirSizeSmaller is like NewDirSizeLarger but returns true if the number
// of entries is smaller or equal than the specified value.
func NewDirSizeSmaller(n, v string) boolexpr.Operand { return DirSizeSmaller(n, v) }

// NewFileSizeLarger returns a boolexpr.Operand that returns true if the expression
// value implements DirSizeIfc and the number of entries in the directory
// is greater than the specified value.
func NewFileSizeLarger(n, v string) boolexpr.Operand { return FileSizeLarger(n, v) }

// NewFileSizeSmaller is like NewFileSizeLarger but returns true if the number
// of entries is smaller or equal than the specified value.
func NewFileSizeSmaller(n, v string) boolexpr.Operand { return FileSizeSmaller(n, v) }

// New returns a boolexpr.Parser with the following operands registered:
//   - "name": case sensitive Glob
//   - "iname", case insensitive Glob
//   - "re", Regxp
//   - "type", FileType
//   - "newer", NewerThan
//   - "dir-larger", DirSizeGreater
//   - "dir-smaller", DirSizeSmaller
func New() *boolexpr.Parser {
	parser := boolexpr.NewParser()
	parser.RegisterOperand("name", NewGlob)
	parser.RegisterOperand("iname", NewIGlob)
	parser.RegisterOperand("re", NewRegexp)
	parser.RegisterOperand("type", NewFileType)
	parser.RegisterOperand("newer", NewNewerThan)
	parser.RegisterOperand("dir-larger", NewDirSizeLarger)
	parser.RegisterOperand("dir-smaller", NewDirSizeSmaller)
	parser.RegisterOperand("file-larger", NewFileSizeLarger)
	parser.RegisterOperand("file-smaller", NewFileSizeSmaller)
	return parser
}
