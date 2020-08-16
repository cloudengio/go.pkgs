package expect_test

import (
	"context"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/expect"
)

func newStream(t *testing.T) (*expect.Stream, *os.File) {
	rd, wr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	return expect.New(rd), wr
}

func writeLines(wr io.Writer, lines ...string) {
	for _, l := range lines {
		wr.Write([]byte(l))
		wr.Write([]byte{'\n'})
	}
}

func assert(t *testing.T, err error) {
	_, _, line, _ := runtime.Caller(1)
	if err != nil {
		t.Errorf("line: %v: %v", line, err)
	}
}

func TestSingle(t *testing.T) {
	st, wr := newStream(t)
	ctx := context.Background()

	go func() {
		time.Sleep(time.Millisecond * 100)
		writeLines(wr, "A", "B", "123-ABC", "1", "2", "3", "AB-77")
		wr.Close()
	}()
	re1, re2 := regexp.MustCompile(`\d+-.*`), regexp.MustCompile(`AB-\d+`)
	assert(t, st.ExpectNext(ctx, "A"))
	assert(t, st.ExpectNext(ctx, "B"))
	assert(t, st.ExpectNextRE(ctx, re1))
	assert(t, st.ExpectEventually(ctx, "3"))
	assert(t, st.ExpectEventuallyRE(ctx, re2))
	assert(t, st.ExpectEOF(ctx))
	assert(t, st.Err())
}

func TestMulti(t *testing.T) {
	st, wr := newStream(t)
	ctx := context.Background()
	go func() {
		time.Sleep(time.Millisecond * 100)
		writeLines(wr, "B", "B", "AB-77", "1", "2", "3", "123-ABC")
		wr.Close()
	}()
	re1, re2 := regexp.MustCompile(`\d+-.*`), regexp.MustCompile(`AB-\d+`)
	assert(t, st.ExpectNext(ctx, "B", "A"))
	assert(t, st.ExpectNext(ctx, "B", "A"))
	assert(t, st.ExpectNextRE(ctx, re1, re2))
	assert(t, st.ExpectNext(ctx, "1", "2"))
	assert(t, st.ExpectNext(ctx, "2", "3"))
	assert(t, st.ExpectNext(ctx, "3"))
	assert(t, st.ExpectEventuallyRE(ctx, re1, re2))
	assert(t, st.ExpectEOF(ctx))
	assert(t, st.Err())
}

func TestErrors(t *testing.T) {
	st, wr := newStream(t)
	ctx := context.Background()

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

	ectx, _ := context.WithTimeout(ctx, time.Millisecond*250)
	err = st.ExpectEOF(ectx)
	expectError("ExpectEOF: failed @ 0:()")
	writeLines(wr, "A B C")
	err = st.ExpectEOF(ctx)
	expectError("ExpectEOF: failed @ 0:()")

}
