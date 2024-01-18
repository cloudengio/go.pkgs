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
		in   interface{}
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
