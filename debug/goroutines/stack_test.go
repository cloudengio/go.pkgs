// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package goroutines

import (
	"strings"
	"testing"
)

var sampleGoroutines = []*Goroutine{
	{
		ID:    1,
		State: "running",
		Stack: []*Frame{
			{
				Call:   "main.main()",
				File:   "/Users/cnicolaou/go/src/main.go",
				Line:   10,
				Offset: 12,
			},
			{
				Call:   "runtime.main()",
				File:   "/usr/local/go/src/runtime/proc.go",
				Line:   250,
				Offset: 0,
			},
		},
		Creator: &Frame{
			Call:   "runtime.startm()",
			File:   "/usr/local/go/src/runtime/proc.go",
			Line:   1123,
			Offset: 34,
		},
	},
	{
		ID:    6,
		State: "chan receive",
		Stack: []*Frame{
			{
				Call:   "main.foobar()",
				File:   "/Users/cnicolaou/go/src/main.go",
				Line:   23,
				Offset: 0,
			},
		},
	},
}

const panicFormat = `goroutine 1 [running]:
	main.main()
	/Users/cnicolaou/go/src/main.go:10 +0xc
	runtime.main()
	/usr/local/go/src/runtime/proc.go:250
created by runtime.startm()
	/usr/local/go/src/runtime/proc.go:1123 +0x22
goroutine 6 [chan receive]:
	main.foobar()
	/Users/cnicolaou/go/src/main.go:23
`

const compactFormat = `goroutine 1 [running]
frame /Users/cnicolaou/go/src/main.go:10 main.main() +0xc
frame /usr/local/go/src/runtime/proc.go:250 runtime.main()
creator /usr/local/go/src/runtime/proc.go:1123 runtime.startm() +0x22
goroutine 6 [chan receive]
frame /Users/cnicolaou/go/src/main.go:23 main.foobar()
`

func TestFormatWithTemplate(t *testing.T) {
	panicT, err := PanicTemplate()
	if err != nil {
		t.Fatal(err)
	}
	out, err := FormatWithTemplate(panicT, sampleGoroutines...)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out, panicFormat; got != want {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}

	compactT, err := CompactTemplate()
	if err != nil {
		t.Fatal(err)
	}
	out, err = FormatWithTemplate(compactT, sampleGoroutines...)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := out, compactFormat; got != want {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}

	_, err = FormatWithTemplate(nil, sampleGoroutines...)
	if err == nil || !strings.Contains(err.Error(), "template is nil") {
		t.Errorf("missing or wrong error for nil template: %v", err)
	}
}
