// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package structdoc_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/structdoc"
)

type S2 struct {
	B float32 `tag:"bar"`
}
type S1 struct {
	A int `tag:"foo"`
	B S2  `tag:"foo-bar"`
}

type S3 struct {
	S1
}

func TestStructDoc(t *testing.T) {
	for i, tc := range []struct {
		in   any
		doc  string
		name string
	}{
		{struct {
			A string `tag:"doc"`
			B string // ignored
		}{}, "detail:\nA: doc\n",
			`struct { A string "tag:\"doc\""; B string }`,
		},
		{struct {
			A string `tag:"doc"`
			B string `tag:"-"`
		}{}, "detail:\nA: doc\n",
			`struct { A string "tag:\"doc\""; B string "tag:\"-\"" }`,
		},
		{struct {
			A string `json:"b" tag:"doc"`
		}{}, "detail:\nb: doc\n",
			`struct { A string "json:\"b\" tag:\"doc\"" }`,
		},
		{struct {
			A string `yaml:"c" tag:"doc"`
		}{}, "detail:\nc: doc\n",
			`struct { A string "yaml:\"c\" tag:\"doc\"" }`,
		},
		{&S1{}, "detail:\nA: foo\nB: foo-bar\n  B: bar\n", "cloudeng.io/cmdutil/structdoc_test.S1"},
		{&S3{}, "detail:\nA: foo\nB: foo-bar\n  B: bar\n", "cloudeng.io/cmdutil/structdoc_test.S3"},
		{struct {
			AShortField               string `tag:"and a short description"`
			ASomewhatLongName         string `tag:"a long description that will wrap to the next line since it's at least 80 chars long"`
			AnEvenLongerNameForAField string `tag:"an even longer description that will wrap to the next line since it's also at least 80 chars long"`
		}{}, `detail:
AShortField:               and a short description
ASomewhatLongName:         a long description that will wrap to the next line
                           since it's at least 80 chars long
AnEvenLongerNameForAField: an even longer description that will wrap to the next
                           line since it's also at least 80 chars long
`,
			`struct { AShortField string "tag:\"and a short description\""; ASomewhatLongName string "tag:\"a long description that will wrap to the next line since it's at least 80 chars long\""; AnEvenLongerNameForAField string "tag:\"an even longer description that will wrap to the next line since it's also at least 80 chars long\"" }`,
		},
		{struct {
			ASomewhatLongName string `tag:"a long description that will wrap to the next line since it's at least 80 chars long"`
			SubStruct         struct {
				AnEvenLongerNameForAField string `tag:"an even longer description that will wrap to the next line since it's also at least 80 chars long"`
			} `tag:"a substruct"`
		}{},
			`detail:
ASomewhatLongName: a long description that will wrap to the next line since it's
                   at least 80 chars long
SubStruct:         a substruct
  AnEvenLongerNameForAField: an even longer description that will wrap to the
                             next line since it's also at least 80 chars long
`,
			`struct { ASomewhatLongName string "tag:\"a long description that will wrap to the next line since it's at least 80 chars long\""; SubStruct struct { AnEvenLongerNameForAField string "tag:\"an even longer description that will wrap to the next line since it's also at least 80 chars long\"" } "tag:\"a substruct\"" }`,
		},
	} {
		desc, err := structdoc.Describe(tc.in, "tag", "detail:\n")
		if err != nil {
			t.Errorf("%v: %v", i, err)
			continue
		}
		if got, want := desc.String(), tc.doc; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
		if got, want := structdoc.TypeName(tc.in), tc.name; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
	_, err := structdoc.Describe(32, "tag", "detail")
	if err == nil || !strings.Contains(err.Error(), "int is not a struct") {
		t.Errorf("unexpected or missing error: %v", err)
	}
}

type R1 struct {
	R []R2 `tag:"R1 recursive"`
}

type R2 struct {
	R []R1 `tag:"R2 recursive"`
}

func TestRecursion(t *testing.T) {
	desc, err := structdoc.Describe(&R1{}, "tag", "detail:\n")
	if err != nil {
		t.Errorf("%v", err)
	}
	if got, want := desc.String(), `detail:
R: []R1 recursive
  R: []R2 recursive
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

type I1 struct {
	A I2 `tag:"I1"`
}

type I2 struct {
	B string `tag:"I2"`
}

func TestIndent(t *testing.T) {
	desc, err := structdoc.Describe(&I1{}, "tag", "detail:\n")
	if err != nil {
		t.Errorf("%v", err)
	}
	if got, want := desc.String(), `detail:
A: I1
  B: I2
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

type P2 struct {
	B string `tag:"P2"`
}

type P3 struct {
	B string `tag:"P3"`
}

type P4 struct {
	B string `tag:"P4"`
}

type P5 struct {
	B string `tag:"P5"`
}

type P1 struct {
	A *P2   `tag:"pointer"`
	C []*P3 `tag:"slice of pointers"`
	D **P4  `tag:"pointer to pointer"`
	E *bool `tag:"a pointer to a non-struct"`
	*P5
}

type Node struct {
	Next *Node `tag:"a self referential pointer"`
	V    int   `tag:"value"`
}

func TestPointers(t *testing.T) {
	desc, err := structdoc.Describe(&P1{}, "tag", "detail:\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := desc.String(), `detail:
A: pointer
  B: P2
C: []slice of pointers
  B: P3
D: pointer to pointer
  B: P4
E: a pointer to a non-struct
B: P5
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// A type that refers to itself via a pointer terminates.
	desc, err = structdoc.Describe(&Node{}, "tag", "detail:\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := desc.String(), `detail:
Next: a self referential pointer
V:    value
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
