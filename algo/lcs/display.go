package lcs

import (
	"fmt"
	"io"
	"strings"
)

func verticalFormatFor(a interface{}) string {
	switch a.(type) {
	case []int64:
		return "% 20d"
	case []int32, []uint8:
		return "%3c"
	case []string:
		return "%s"
	default:
		panic(fmt.Sprintf("unsupported type: %T", a))
	}
}

// FormatVertical prints a representation of the edit script with one
// item per line, eg:
//   -  6864772235558415538
//     -8997218578518345818
//   + -6615550055289275125
//   - -7192184552745107772
//      5717881983045765875
func FormatVertical(out io.Writer, a interface{}, script EditScript) {
	lookupA := accessorFor(a)
	format := verticalFormatFor(a)
	for _, op := range script {
		switch op.Op {
		case Identical:
			f := fmt.Sprintf(format, lookupA(op.A))
			out.Write([]byte(fmt.Sprintf("  %s\n", f)))
		case Delete:
			f := fmt.Sprintf(format, lookupA(op.A))
			out.Write([]byte(fmt.Sprintf("- %s\n", f)))
		case Insert:
			f := fmt.Sprintf(format, op.Val)
			out.Write([]byte(fmt.Sprintf("+ %s\n", f)))
		}
	}
}

func horizontalFormatFor(a interface{}) string {
	switch a.(type) {
	case []int64:
		return "%d"
	case []int32, []uint8:
		return "%c"
	case []string:
		return "%s"
	default:
		panic(fmt.Sprintf("unsupported type: %T", a))
	}
}

// FormatVertical prints a representation of the edit script across
// three lines, with the top line showing the result of applying the
// edit, the middle line the operations applied and the bottom line
// any items deleted, eg:
//   CB AB AC
//  -+|-||-|+
//  A  C  B
func FormatHorizontal(out io.Writer, a interface{}, script EditScript) {
	lookupA := accessorFor(a)
	format := horizontalFormatFor(a)
	displaySizes := []int{}
	for _, op := range script {
		var f string
		switch op.Op {
		case Identical:
			f = fmt.Sprintf(format, lookupA(op.A))
			out.Write([]byte(f))
		case Delete:
			f = fmt.Sprintf(format, lookupA(op.A))
			out.Write([]byte(strings.Repeat(" ", len(f))))
		case Insert:
			f = fmt.Sprintf(format, op.Val)
			out.Write([]byte(f))
		}
		displaySizes = append(displaySizes, len(f))
	}
	out.Write([]byte{'\n'})

	pad := func(o string, i int) {
		out.Write([]byte(o))
		out.Write([]byte(strings.Repeat(" ", displaySizes[i]-len(o))))
	}

	for i, op := range script {
		switch op.Op {
		case Identical:
			pad("|", i)
		case Delete:
			pad("-", i)
		case Insert:
			pad("+", i)
		}
	}
	out.Write([]byte{'\n'})
	for i, op := range script {
		switch op.Op {
		case Delete:
			f := fmt.Sprintf(format, lookupA(op.A))
			out.Write([]byte(f))
		default:
			pad("", i)
		}
	}
	out.Write([]byte{'\n'})
}
