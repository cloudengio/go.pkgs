// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package expect_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/expect"
)

func ExampleLines() {
	ctx := context.Background()
	rd, wr, _ := os.Pipe()
	st := expect.NewLineStream(rd)

	go func() {
		fmt.Fprintf(wr, "A\nready\nC\n")
		wr.Close()
	}()
	_ = st.ExpectEventually(ctx, "ready")
	fmt.Println(st.LastMatch())
	_ = st.ExpectNext(ctx, "C")
	fmt.Println(st.LastMatch())
	_ = st.ExpectEOF(ctx)
	if err := st.Err(); err != nil {
		panic(err)
	}
	// Output:
	// 2 ready
	// 3 C
}

func newLineStream(t *testing.T) (*expect.Lines, *os.File) {
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	return expect.NewLineStream(rd), wr
}

func writeLines(wr io.Writer, lines ...string) {
	for _, l := range lines {
		if _, err := wr.Write([]byte(l)); err != nil {
			panic(err)
		}
		if _, err := wr.Write([]byte{'\n'}); err != nil {
			panic(err)
		}
	}
}

func assert(t *testing.T, err error) {
	_, _, line, _ := runtime.Caller(1)
	if err != nil {
		t.Errorf("line: %v: %v", line, err)
	}
}

func assertMatch(t *testing.T, st *expect.Lines, input string, line int) {
	_, _, loc, _ := runtime.Caller(1)
	n, c := st.LastMatch()
	if got, want := c, input; got != want {
		t.Errorf("line: %v: got %v, want %v", loc, got, want)
	}
	if got, want := n, line; got != want {
		t.Errorf("line: %v: got %v, want %v", loc, got, want)
	}
}

type errorReader struct{}

func (er *errorReader) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf("an error")
}

func TestSingle(t *testing.T) {
	ctx := context.Background()
	st, wr := newLineStream(t)

	go func() {
		time.Sleep(time.Millisecond * 100)
		writeLines(wr, "A", "B", "123-ABC", "1", "2", "3", "AB-77")
		wr.Close()
	}()
	re1, re2 := regexp.MustCompile(`\d+-.*`), regexp.MustCompile(`AB-\d+`)
	assert(t, st.ExpectNext(ctx, "A"))
	assertMatch(t, st, "A", 1)
	assert(t, st.ExpectNext(ctx, "B"))
	assertMatch(t, st, "B", 2)
	assert(t, st.ExpectNextRE(ctx, re1))
	assertMatch(t, st, "123-ABC", 3)
	assert(t, st.ExpectEventually(ctx, "3"))
	assertMatch(t, st, "3", 6)
	assert(t, st.ExpectEventuallyRE(ctx, re2))
	assertMatch(t, st, "AB-77", 7)
	assert(t, st.ExpectEOF(ctx))
	assertMatch(t, st, "AB-77", 7)
	assert(t, st.Err())
}

func TestMulti(t *testing.T) {
	ctx := context.Background()
	st, wr := newLineStream(t)

	go func() {
		time.Sleep(time.Millisecond * 100)
		writeLines(wr, "B", "B", "AB-77", "1", "2", "3", "123-ABC")
		wr.Close()
	}()
	re1, re2 := regexp.MustCompile(`\d+-.*`), regexp.MustCompile(`AB-\d+`)
	assert(t, st.ExpectNext(ctx, "B", "A"))
	assertMatch(t, st, "B", 1)
	assert(t, st.ExpectNext(ctx, "B", "A"))
	assertMatch(t, st, "B", 2)
	assert(t, st.ExpectNextRE(ctx, re1, re2))
	assertMatch(t, st, "AB-77", 3)
	assert(t, st.ExpectNext(ctx, "1", "2"))
	assert(t, st.ExpectNext(ctx, "2", "3"))
	assert(t, st.ExpectNext(ctx, "3"))
	assert(t, st.ExpectEventuallyRE(ctx, re1, re2))
	assert(t, st.ExpectEOF(ctx))
	assert(t, st.Err())
}

func TestErrors(t *testing.T) {
	ctx := context.Background()
	st, wr := newLineStream(t)

	var err error
	expectError := func(want string) {
		_, _, line, _ := runtime.Caller(1)
		if err == nil {
			t.Errorf("line: %v: expected an error", line)
			return
		}
		if got := err.Error(); !strings.Contains(got, want) {
			t.Errorf("line: %v: error %v does not contain %v", line, got, want)
		}
	}
	re1, re2 := regexp.MustCompile(`\d+-.*`), regexp.MustCompile(`AB-\d+`)
	ectx, cancel := context.WithTimeout(ctx, time.Millisecond*250)
	defer cancel()
	err = st.ExpectEOF(ectx)
	expectError("ExpectEOF: failed @ 0: context deadline exceeded")
	writeLines(wr, "A B C")
	assert(t, st.ExpectNext(ctx, "A B C"))
	writeLines(wr, "A B C")
	err = st.ExpectNext(ctx, "A B")
	expectError("ExpectNext: failed @ 2:\nA B C\n!=\nA B")
	writeLines(wr, "A B C")
	err = st.ExpectNext(ctx, "A", "B")
	expectError("ExpectNext: failed @ 3:\nA B C\n!= any of:\nA\nB")
	writeLines(wr, "A B C")
	err = st.ExpectNextRE(ctx, re1)
	expectError("ExpectNextRE: failed @ 4:\nA B C\n!=\n\\d+-.*")
	writeLines(wr, "A B C")
	err = st.ExpectNextRE(ctx, re1, re2)
	expectError("ExpectNextRE: failed @ 5:\nA B C\n!= any of:\n\\d+-.*\nAB-\\d+")
	writeLines(wr, "A B C")
	ectx, cancel = context.WithTimeout(ctx, time.Millisecond*250)
	defer cancel()
	err = st.ExpectEventuallyRE(ectx, re1, re2)
	expectError("ExpectEventuallyRE: failed @ 6: context deadline exceeded")

	writeLines(wr, "A B C")
	ectx, cancel = context.WithTimeout(ctx, time.Millisecond*250)
	defer cancel()
	err = st.ExpectEventually(ectx, "A", "B")
	expectError("ExpectEventually: failed @ 7: context deadline exceeded")

	writeLines(wr, "A B C")
	err = st.ExpectEOF(ctx)
	expectError("ExpectEOF: failed @ 8:\nA B C\n!=\n<EOF>")

	wr.Close()
	err = st.ExpectNext(ctx, "A", "B")
	expectError("ExpectNext: failed @ 8:\n<EOF>\n!= any of:\nA\nB")
	err = st.ExpectNext(ctx, "A", "B")
	expectError("ExpectNext: failed @ 8:\n<EOF>\n!= any of:\nA\nB")
	err = st.ExpectEventually(ctx, "A", "B")
	expectError("ExpectEventually: failed @ 8:\n<EOF>\n!= any of:\nA\nB")
	err = st.ExpectNextRE(ctx, re1)
	expectError("ExpectNextRE: failed @ 8:\n<EOF>\n!=\n\\d+-.*")

	erd := &errorReader{}
	st = expect.NewLineStream(erd)
	err = st.ExpectNext(ctx, "A", "B")
	expectError("ExpectNext: failed @ 0: an error")
	st = expect.NewLineStream(erd)
	err = st.ExpectNextRE(ctx, re1)
	expectError("ExpectNextRE: failed @ 0: an error")
	st = expect.NewLineStream(erd)
	err = st.ExpectEventually(ctx, "A", "B")
	expectError("ExpectEventually: failed @ 0: an error")
}
