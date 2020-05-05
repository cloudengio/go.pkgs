package lcs

import (
	"fmt"
	"io"
)

// PrettyVertical prints a representation as an edit of the original
// slice. The script must be obtained via ReplayScript.
func PrettyVertical(out io.Writer, a interface{}, script EditScript) {
	lookupA := accessorFor(a)
	format := fmtFor(a)
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
