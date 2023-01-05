// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package goroutines

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

var goroutineHeaderRE = regexp.MustCompile(`^goroutine (\d+) \[([^\]]+)\]:$`)

// Goroutine represents a single goroutine.
type Goroutine struct {
	ID      int64
	State   string
	Stack   []*Frame
	Creator *Frame
}

// Get gets a set of currently running goroutines and parses them into a
// structured representation. Any goroutines that match the ignore list are
// ignored.
func Get(ignore ...string) ([]*Goroutine, error) {
	bufsize, read := 1<<20, 0
	buf := make([]byte, bufsize)
	for {
		read = runtime.Stack(buf, true)
		if read < bufsize {
			buf = buf[:read]
			break
		}
		bufsize *= 2
		buf = make([]byte, bufsize)
	}
	return Parse(buf, ignore...)
}

// Parse parses a stack trace into a structure representation.
func Parse(buf []byte, ignore ...string) ([]*Goroutine, error) {
	scanner := bufio.NewScanner(bytes.NewReader(buf))
	var out []*Goroutine
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		g, err := parseGoroutine(scanner)
		if err != nil {
			return out, fmt.Errorf("Error parsing trace: %v\n%s", err, string(buf))
		}
		if !shouldIgnore(g, ignore) {
			out = append(out, g)
		}
	}
	return out, scanner.Err()
}

func shouldIgnore(g *Goroutine, ignoredGoroutines []string) bool {
	for _, ignored := range ignoredGoroutines {
		if c := g.Creator; c != nil && strings.Contains(c.Call, ignored) {
			return true
		}
		for _, f := range g.Stack {
			if strings.Contains(f.Call, ignored) {
				return true
			}
		}
	}
	return false
}

func parseGoroutine(scanner *bufio.Scanner) (*Goroutine, error) {
	g := &Goroutine{}
	matches := goroutineHeaderRE.FindSubmatch(scanner.Bytes())
	if len(matches) != 3 {
		return nil, fmt.Errorf("Could not parse goroutine header from: %s", scanner.Text())
	}
	id, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		return nil, err
	}
	g.ID = id
	g.State = string(matches[2])

	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			break
		}
		frame, err := parseFrame(scanner)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(frame.Call, "created by ") {
			frame.Call = frame.Call[len("created by "):]
			g.Creator = frame
			break
		}
		g.Stack = append(g.Stack, frame)
	}
	return g, nil
}

func (g *Goroutine) writeTo(w io.Writer) {
	fmt.Fprintf(w, "goroutine %d [%s]:\n", g.ID, g.State)
	for _, f := range g.Stack {
		f.writeTo(w)
	}
	if g.Creator != nil {
		fmt.Fprint(w, "created by ")
		g.Creator.writeTo(w)
	}
}

// Frame represents a single stack frame.
type Frame struct {
	Call   string
	File   string
	Line   int64
	Offset int64
}

func parseFrame(scanner *bufio.Scanner) (*Frame, error) {
	f := &Frame{Call: scanner.Text()}
	if !scanner.Scan() {
		return nil, fmt.Errorf("Frame lacked a second line %s", f.Call)
	}
	var err error
	f.File, f.Line, f.Offset, err = parseFileLine(scanner.Bytes())
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Frame) writeTo(w io.Writer) {
	fmt.Fprintln(w, f.Call)
	if f.Offset != 0 {
		fmt.Fprintf(w, "\t%s:%d +0x%x\n", f.File, f.Line, f.Offset)
	} else {
		fmt.Fprintf(w, "\t%s:%d\n", f.File, f.Line)
	}
}

// Format formats Goroutines back into the normal string representation.
func Format(gs ...*Goroutine) string {
	out := &strings.Builder{}
	for i, g := range gs {
		if i != 0 {
			out.WriteRune('\n')
		}
		g.writeTo(out)
	}
	return out.String()
}
