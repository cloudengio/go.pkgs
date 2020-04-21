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
		}{}, "detail:\nA: doc\n",
			`struct { A string "tag:\"doc\""; B string }`,
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
