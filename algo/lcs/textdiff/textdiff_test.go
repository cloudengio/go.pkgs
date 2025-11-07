// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textdiff_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/algo/lcs/textdiff"
)

func TestDiffGroups(t *testing.T) {
	l := func(s ...string) []string { return s }
	for e, engine := range []func(a, b []byte) *textdiff.Diff{
		textdiff.LinesDP, textdiff.LinesMyers,
	} {
		for i, tc := range []struct {
			a, b    string
			summary []string
		}{
			{"S\nA\nB\nC\nD\nE", "S\nC", l("2,3d1", "5,6d2")},
			{"S\nA\nB\nC\nD\nE", "C", l("1,3d0", "5,6d1")},
			{"A\nB\nC\nD\nE", "C", l("1,2d0", "4,5d1")},
			{"S\nC", "S\nA\nB\nC\nD\nE", l("1a2,3", "2a5,6")},
			{"C", "S\nA\nB\nC\nD\nE", l("0a1,3", "1a5,6")},
			{"C", "A\nB\nC\nD\nE", l("0a1,2", "1a4,5")},
			{"S\nA\nB\nC\nD\nE", "S\nS\nC", l("2,3c2", "5,6d3")},
			{"S\nA\nB\nC\nS", "S\nAA\nBB\nCC\nS", l("2,4c2,4")},
			{"S\nAA\nBB\nCC\nS", "S\nA\nB\nC\nS", l("2,4c2,4")},
		} {
			diffs := engine([]byte(tc.a), []byte(tc.b))
			ng := diffs.NumGroups()
			if got, want := ng, len(tc.summary); got != want {
				t.Errorf("%v.%v: got %v, want %v\n", e, i, got, want)
				continue
			}
			for g := 0; g < ng; g++ {
				if got, want := diffs.Group(g).Summary(), tc.summary[g]; got != want {
					t.Errorf("%v.%v: got %v, want %v\n", e, i, got, want)
				}
			}
		}
	}
}

func processDiffOutput(t *testing.T, diffFile string) (inserted, deleted []string) {
	diffs, err := os.Open(diffFile)
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(diffs)

	inRun := false
	var deletedText, insertedText string
	appendRun := func() {
		if !inRun {
			return
		}
		inserted = append(inserted, insertedText)
		deleted = append(deleted, deletedText)
		insertedText, deletedText = "", ""
		inRun = true
	}
	for sc.Scan() {
		l := sc.Text()
		if len(l) == 0 {
			appendRun()
			continue
		}
		switch l[0] {
		case '>':
			insertedText += l[2:] + "\n"
			inRun = true
		case '<':
			deletedText += l[2:] + "\n"
			inRun = true
		case '-':
		default:
			appendRun()
		}
	}
	appendRun()
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return
}

var (
	diffOutput = []string{
		"7a8,9",
		"15,17d16",
		"31a31,42",
		"37a49",
		"118a131",
		"132c145",
		"137,139c150,151",
	}
)

func TestTextDiff(t *testing.T) {
	f1, f2 := filepath.Join("testdata", "textdiff.go.a"), filepath.Join("testdata", "textdiff.go.b")
	a, err := os.ReadFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f2)
	if err != nil {
		t.Fatal(err)
	}

	// The diff output may differ in detail but be equivalent
	// since the edits can be ambgious.
	dpOutput := append([]string{}, diffOutput...)
	dpOutput[1] = "14,16d15"
	dpOutput[2] = "30a30,41"

	myersOutput := append([]string{}, diffOutput...)
	myersOutput[2] = dpOutput[2]

	insertedAll, deletedAll := processDiffOutput(t, filepath.Join("testdata", "textdiff.go.a.b"))

	//	insertedAll[0] = strings.TrimPrefix(insertedAll[0], "\n") + "\n"
	deletedAll[1] = "\n" + strings.TrimSuffix(deletedAll[1], "\n")
	insertedAll[2] = "\n" + strings.TrimSuffix(insertedAll[2], "\n")

	for e, tc := range []struct {
		engine func(a, b []byte) *textdiff.Diff
		output []string
	}{
		{textdiff.LinesDP, dpOutput},
		{textdiff.LinesMyers, myersOutput},
	} {
		if e != 0 {
			continue
		}
		diffs := tc.engine(a, b)
		if got, want := diffs.NumGroups(), len(tc.output); got != want {
			t.Errorf("%v: got %v, want %v", e, got, want)
		}
		for i := 0; i < diffs.NumGroups(); i++ {
			dg := diffs.Group(i)
			if got, want := dg.Summary(), tc.output[i]; got != want {
				t.Errorf("%v.%v: got %v, want %v", e, i, got, want)
				t.Logf(" got: % 02x\n", got)
				t.Logf("want: % 02x\n", want)
			}
			if got, want := dg.Inserted(), insertedAll[i]; got != want {
				t.Errorf("%v.%v: got __%v__, want __%v__", e, i, got, want)
				t.Logf(" got: % 02x\n", got)
				t.Logf("want: % 02x\n", want)
			}
			if got, want := dg.Deleted(), deletedAll[i]; got != want {
				t.Errorf("%v.%v: got %v, want %v", e, i, got, want)
				t.Logf(" got: % 02x\n", got)
				t.Logf("want: % 02x\n", want)
			}
		}
	}
}
