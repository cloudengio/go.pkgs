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
	"text/template"
)

var goroutineHeaderRE = regexp.MustCompile(`^goroutine (\d+) \[([^\]]+)\]:$`)

const (
	panicTemplateText = `{{if .IsHeader }}goroutine {{.Goroutine.ID}} [{{.Goroutine.State}}]:
{{end}}{{if .IsFrame}}{{.Frame.Call}}
    {{.Frame.File}}:{{.Frame.Line}}{{if .HasOffset}} +0x{{.OffsetHex}}{{end}}
{{end}}{{if .IsCreator}}created by {{.Frame.Call}}
    {{.Frame.File}}:{{.Frame.Line}}{{if .HasOffset}} +0x{{.OffsetHex}}
{{end}}
{{end}}`

	compactTemplateText = `{{if .IsHeader }}goroutine {{.Goroutine.ID}} [{{.Goroutine.State}}]:
{{end}}{{if .IsFrame}}    {{.Frame.File}}:{{.Frame.Line}} {{.Frame.Call}}
{{end}}{{if .IsCreator}}    created by {{.Frame.Call}} {{.Frame.File}}:{{.Frame.Line}}

{{end}}`
)

var (
	panicTemplateCompiled   = template.Must(template.New("goroutines_panic").Parse(panicTemplateText))
	compactTemplateCompiled = template.Must(template.New("goroutines_compact").Parse(compactTemplateText))
)

// PanicTemplate returns a template that mimics the formatting produced by a
// Go panic stack trace. The returned template is a clone of an internal
// instance, so callers may modify it without affecting future calls.
func PanicTemplate() (*template.Template, error) {
	return panicTemplateCompiled.Clone()
}

// CompactTemplate returns a single-line-per-frame representation template that
// emits concise goroutine information.
func CompactTemplate() (*template.Template, error) {
	return compactTemplateCompiled.Clone()
}

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
			return out, fmt.Errorf("error parsing trace: %v\n%s", err, string(buf))
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
		return nil, fmt.Errorf("could not parse goroutine header from: %s", scanner.Text())
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
		return nil, fmt.Errorf("frame lacked a second line %s", f.Call)
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

// TemplateData provides the context passed to templates executed by
// FormatWithTemplate. Fields are exported so they can be accessed from the
// template, including convenience booleans describing the position of the
// current line.
type TemplateData struct {
	Goroutine *Goroutine
	Frame     *Frame

	GoroutineIndex int
	GoroutineCount int
	FrameIndex     int
	FrameCount     int

	IsHeader         bool
	IsFrame          bool
	IsCreator        bool
	IsFirstGoroutine bool
	IsLastGoroutine  bool
	IsFirstFrame     bool
	IsLastFrame      bool
	HasFrames        bool
	HasCreator       bool
	HasOffset        bool
	OffsetHex        string
}

// FormatWithTemplate renders a collection of goroutines using the supplied
// template. The template is executed once for each line that would appear in the
// textual stack trace: the goroutine header, each frame, and the optional
// creator frame. The provided TemplateData exposes the raw goroutine/frame
// values along with helper booleans that enable conditional formatting from the
// template itself.
func FormatWithTemplate(tmpl *template.Template, gs ...*Goroutine) (string, error) {
	if tmpl == nil {
		return "", fmt.Errorf("goroutines: template is nil")
	}
	var buf bytes.Buffer
	total := len(gs)

	for gi, g := range gs {
		data := TemplateData{
			Goroutine:        g,
			GoroutineIndex:   gi,
			GoroutineCount:   total,
			FrameIndex:       -1,
			FrameCount:       len(g.Stack),
			IsHeader:         true,
			HasFrames:        len(g.Stack) > 0,
			HasCreator:       g.Creator != nil,
			IsFirstGoroutine: gi == 0,
			IsLastGoroutine:  gi == total-1,
		}
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}

		for fi, frame := range g.Stack {
			data = TemplateData{
				Goroutine:        g,
				Frame:            frame,
				GoroutineIndex:   gi,
				GoroutineCount:   total,
				FrameIndex:       fi,
				FrameCount:       len(g.Stack),
				IsFrame:          true,
				HasFrames:        len(g.Stack) > 0,
				HasCreator:       g.Creator != nil,
				IsFirstGoroutine: gi == 0,
				IsLastGoroutine:  gi == total-1,
				IsFirstFrame:     fi == 0,
				IsLastFrame:      fi == len(g.Stack)-1,
				HasOffset:        frame.Offset != 0,
				OffsetHex:        fmt.Sprintf("%x", frame.Offset),
			}
			if err := tmpl.Execute(&buf, data); err != nil {
				return "", err
			}
		}

		if g.Creator != nil {
			data = TemplateData{
				Goroutine:        g,
				Frame:            g.Creator,
				GoroutineIndex:   gi,
				GoroutineCount:   total,
				FrameIndex:       len(g.Stack),
				FrameCount:       len(g.Stack),
				IsCreator:        true,
				HasFrames:        len(g.Stack) > 0,
				HasCreator:       true,
				IsFirstGoroutine: gi == 0,
				IsLastGoroutine:  gi == total-1,
				HasOffset:        g.Creator.Offset != 0,
				OffsetHex:        fmt.Sprintf("%x", g.Creator.Offset),
			}
			if err := tmpl.Execute(&buf, data); err != nil {
				return "", err
			}
		}
	}

	return buf.String(), nil
}
