package flags

import (
	"fmt"
	"strings"
)

// ErrInvalidRange represents the error generated for an invalid
// range. Use errors.Is to test for it.
type ErrInvalidRange struct {
	msg string
}

// Error implements error.
func (ire *ErrInvalidRange) Error() string {
	return ire.msg
}

// Is implements errors.Is.
func (ire ErrInvalidRange) Is(target error) bool {
	_, ok := target.(*ErrInvalidRange)
	return ok
}

// RangeSpec represents a specification for a 'range' such as that used
// to specify pages to be printed or table columns to be accessed.
// It implements flag.Value.
//
// Each range is of the general form:
//
//    <from>[-<to>] | -<from>[-<to>|-] | <from>-
//
// which allows for the following:
//   <from>        : a single item
//   <from>-<to>   : a range of one or more items
//   -<from>       : a single item, relative to the end
//   -<from>-<to>  : a range, whose start and end are indexed relative the end
//   -<from>-      : a range, relative to the end that extends to the end
//   <from>-       : a range that extends to the end
//
// Note that the interpretation of these ranges is left to users of this
// type. For example, intepreting these values as pages in a document could
// lead to the following:
//
//   3      : page 3
//  2-4     : pages 2 through 4
//  4-2     : pages 4 through 2
//   -2     : second to last page
//  -4-2    : fourth from last to second from last
//  -2-4    : second from last to fourth from last
//  -2-     : second to last and all following pages
//  2-      : page 2 and all following pages.
type RangeSpec struct {
	From, To      string
	RelativeToEnd bool
	ExtendsToEnd  bool
}

func (rs RangeSpec) writeString(sep string, out *strings.Builder) {
	if rs.RelativeToEnd {
		out.WriteString(sep)
	}
	out.WriteString(rs.From)
	if len(rs.To) > 0 {
		out.WriteString(sep)
		out.WriteString(rs.To)
	}
	if rs.ExtendsToEnd {
		out.WriteString(sep)
	}
}

// String implements flag.Value.
func (rs RangeSpec) String() string {
	out := &strings.Builder{}
	rs.writeString("-", out)
	return out.String()
}

// Set implements flag.Value.
func (rs *RangeSpec) Set(v string) error {
	return rs.set('-', v)
}

func (rs *RangeSpec) set(sep byte, v string) error {
	if strings.Count(v, ",") > 0 {
		return &ErrInvalidRange{msg: fmt.Sprintf("invalid range: contains a ,: %v", v)}
	}
	spec, err := parseRangeSpec(sep, v)
	if err != nil {
		return err
	}
	*rs = spec
	return nil
}

func parseSingleDash(sep byte, val string) (RangeSpec, error) {
	if len(val) == 1 {
		return RangeSpec{}, &ErrInvalidRange{msg: "invalid range: empty range"}
	}
	idx := strings.IndexByte(val, sep)
	switch {
	case idx == 0:
		// -<from>
		return RangeSpec{From: val[1:], RelativeToEnd: true}, nil
	case idx == len(val)-1:
		// <from>-
		return RangeSpec{From: val[:len(val)-1], ExtendsToEnd: true}, nil
	}
	// <from>-<to>
	return RangeSpec{From: val[:idx], To: val[idx+1:]}, nil
}

func parseDoubleDash(sep byte, val string) (RangeSpec, error) {
	idx := strings.IndexByte(val, sep)
	ridx := strings.LastIndexByte(val, sep)
	switch {
	case idx+1 == ridx:
		// --
		return RangeSpec{}, &ErrInvalidRange{msg: fmt.Sprintf("invalid range, empty range: %v", val)}
	case idx == 0 && ridx == len(val)-1:
		// -<from>-
		return RangeSpec{From: val[1 : len(val)-1], RelativeToEnd: true, ExtendsToEnd: true}, nil
	case idx == 0:
		// -<from>-<to>
		return RangeSpec{From: val[1:ridx], To: val[ridx+1:], RelativeToEnd: true}, nil
	default:
		// eg: a-b-c.
		return RangeSpec{}, &ErrInvalidRange{msg: fmt.Sprintf("invalid range: %v", val)}
	}
}

func parseRangeSpec(sep byte, val string) (RangeSpec, error) {
	if len(val) == 0 {
		return RangeSpec{}, &ErrInvalidRange{msg: "invalid range: empty range"}
	}
	dashes := strings.Count(val, string(sep))
	switch {
	case dashes == 0:
		return RangeSpec{From: val}, nil
	case dashes == 1:
		return parseSingleDash(sep, val)
	case dashes == 2:
		return parseDoubleDash(sep, val)
	default:
		return RangeSpec{}, &ErrInvalidRange{msg: fmt.Sprintf("invalid range, too many %c's: %v", sep, val)}
	}
}

// RangeSpecs represents a slice of RangeSpec. It implements
// flag.Value
type RangeSpecs []RangeSpec

// Set implements flag.Value.
func (rs *RangeSpecs) Set(val string) error {
	for _, p := range strings.Split(val, ",") {
		rg, err := parseRangeSpec('-', p)
		if err != nil {
			return err
		}
		*rs = append(*rs, rg)
	}
	return nil
}

// String implements flag.Value.
func (rs *RangeSpecs) String() string {
	out := &strings.Builder{}
	rl := len((*rs)) - 1
	for i, s := range *rs {
		s.writeString("-", out)
		if i < rl {
			out.WriteString(",")
		}
	}
	return out.String()
}

// ColonRangeSpec is like RangeSpec except that : is the separator.
type ColonRangeSpec struct {
	RangeSpec
}

// String implements flag.Value.
func (crs *ColonRangeSpec) String() string {
	out := &strings.Builder{}
	crs.writeString(":", out)
	return out.String()
}

// Set implements flag.Value.
func (crs *ColonRangeSpec) Set(v string) error {
	return crs.set(':', v)
}

// ColonRangeSpecs represents a slice of ColonRangeSpec. It implements
// flag.Value.
type ColonRangeSpecs []ColonRangeSpec

// Set implements flag.Value.
func (crs *ColonRangeSpecs) Set(val string) error {
	for _, p := range strings.Split(val, ",") {
		crg, err := parseRangeSpec(':', p)
		if err != nil {
			return err
		}
		*crs = append(*crs, ColonRangeSpec{crg})
	}
	return nil
}

// String implements flag.Value.
func (crs *ColonRangeSpecs) String() string {
	out := &strings.Builder{}
	crl := len((*crs)) - 1
	for i, s := range *crs {
		s.writeString(":", out)
		if i < crl {
			out.WriteString(",")
		}
	}
	return out.String()
}
