package textdiff_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"cloudeng.io/algo/lcs/textdiff"
)

func TestTextDiff(t *testing.T) {
	a, err := ioutil.ReadFile(filepath.Join("testdata", "textdiff.go.a"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadFile(filepath.Join("testdata", "textdiff.go.b"))
	if err != nil {
		t.Fatal(err)
	}
	diffs := textdiff.DiffByLines(a, b)
	for i := 0; i < diffs.NumGroups(); i++ {
		dg := diffs.Group(i)
		fmt.Printf("%v === %v\n", i, dg.Summary())
	}
	//t.Log(diffs)
	_ = diffs
	t.Fail()

}
