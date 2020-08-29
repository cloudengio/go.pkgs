package flags

import (
	"flag"
	"strings"
)

// Defaults returns the output of PrintDefaults() as a string.
func Defaults(fs *flag.FlagSet) string {
	out := &strings.Builder{}
	orig := fs.Output()
	defer fs.SetOutput(orig)
	fs.SetOutput(out)
	fs.PrintDefaults()
	return out.String()
}
