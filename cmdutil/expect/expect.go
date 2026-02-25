// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package expect provides support for making expectations on the contents
// of input streams.
package expect

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"cloudeng.io/errors"
)

// UnexpectedInputError represents a failed expectation, i.e. when the contents
// of the input do not match the expected contents.
type UnexpectedInputError struct {
	Err         error            // An underlying error, if any, eg. context cancelation.
	Line        int              // The number of the last line that was read.
	Input       string           // The last input line that was read.
	EOF         bool             // Set if EOF was encountered.
	Eventually  bool             // Set if the input was 'eventually' expected.
	EOFExpected bool             // Set if EOF was expected.
	Literals    []string         // The literal strings that were expected.
	Expressions []*regexp.Regexp // The regular sxpressiones that were expected.
}

func (e *UnexpectedInputError) Expectation() string {
	if e.EOFExpected {
		return "ExpectEOF"
	}
	lit := len(e.Literals) > 0
	if e.Eventually {
		if lit {
			return "ExpectEventually"
		}
		return "ExpectEventuallyRE"
	}
	if lit {
		return "ExpectNext"
	}
	return "ExpectNextRE"
}

func (e *UnexpectedInputError) formatLiterals(out io.Writer) {
	switch len(e.Literals) {
	case 0:
	case 1:
		fmt.Fprintf(out, "\n!=\n%s", e.Literals[0])
	default:
		fmt.Fprintf(out, "\n!= any of:\n")
		for _, l := range e.Literals {
			fmt.Fprintf(out, "%s\n", l)
		}
	}
}

func (e *UnexpectedInputError) formatREs(out io.Writer) {
	switch len(e.Expressions) {
	case 0:
	case 1:
		fmt.Fprintf(out, "\n!=\n%s", e.Expressions[0])
	default:
		fmt.Fprintf(out, "\n!= any of:\n")
		for _, re := range e.Expressions {
			fmt.Fprintf(out, "%s\n", re)
		}
	}
}

// Error implements error.
func (e *UnexpectedInputError) Error() string {
	opname := e.Expectation()
	if err := e.Err; err != nil {
		return fmt.Sprintf("%s: failed @ %v: %v", opname, e.Line, err)
	}
	out := &strings.Builder{}
	input := e.Input
	if e.EOF {
		input = "<EOF>"
	}
	fmt.Fprintf(out, "%s: failed @ %v:\n%v", opname, e.Line, input)
	if e.EOFExpected {
		fmt.Fprintf(out, "\n!=\n<EOF>")
	} else {
		e.formatLiterals(out)
		e.formatREs(out)
	}
	return out.String()
}

// Lines provides line oriented expecations and will block waiting for the
// expected input. A context with a timeout or deadline can be used to abort
// the expectation. Literal and regular expression matches are supported as is
// matching on EOF. Each operation accepts multiple literals or regular
// expressions that are treated as an 'or' to allow for convenient handling
// of different input orderings.
type Lines struct {
	rd        io.Reader
	ch        chan *inputEvent
	eof       bool
	input     string
	line      int
	lastMatch string
	lastLine  int
	options   options
	errs      *errors.M
}

type options struct {
	trace io.Writer
}

// Option represents an option.
type Option func(*options)

// TraceInput enables tracing of input as it is read.
func TraceInput(out io.Writer) Option {
	return func(o *options) {
		o.trace = out
	}
}

// NewLineStream creates a new instance of Lines.
func NewLineStream(rd io.Reader, opts ...Option) *Lines {
	s := &Lines{
		rd:   rd,
		ch:   make(chan *inputEvent, 1),
		errs: &errors.M{},
	}
	for _, fn := range opts {
		fn(&s.options)
	}
	go readLines(rd, s.options.trace, s.ch)
	return s
}

type inputEvent struct {
	input string
	err   error
}

func readLines(rd io.Reader, out io.Writer, ch chan<- *inputEvent) {
	brd := bufio.NewReader(rd)
	defer close(ch)
	for {
		str, err := brd.ReadString('\n')
		if out != nil {
			fmt.Fprintf(out, "> %s", str) //nolint:gosec // G705: XSS via taint analysis
		}
		if err != nil {
			if err == io.EOF {
				// closing the chanel indicates EOF.
				return
			}
			ch <- &inputEvent{err: err}
			return
		}
		ch <- &inputEvent{
			input: strings.TrimSuffix(str, "\n"),
		}
	}
}

// Err returns all errors encountered. Note that closing the underlying io.Reader
// is not considered an error unless ExpectEOF failed.
func (s *Lines) Err() error {
	return s.errs.Err()
}

func (s *Lines) nextLine(ctx context.Context) error {
	select {
	case rec := <-s.ch:
		switch {
		case rec == nil:
			s.eof = true
			return nil
		case rec.err != nil:
			return &UnexpectedInputError{
				Err:   rec.err,
				Line:  s.line,
				EOF:   s.eof,
				Input: s.input,
			}
		}
		s.input = rec.input
		s.line++
	case <-ctx.Done():
		return &UnexpectedInputError{
			Err:   ctx.Err(),
			EOF:   s.eof,
			Line:  s.line,
			Input: s.input,
		}
	}
	return nil
}

// ExpectNext will return nil if one of the supplied lines is equal to the next
// line read from the input stream. It will block waiting for the next line;
// the supplied context can be used to provide a timeout.
func (s *Lines) ExpectNext(ctx context.Context, lines ...string) error {
	if err := s.nextLine(ctx); err != nil {
		err.(*UnexpectedInputError).Literals = lines
		s.errs.Append(err)
		return err
	}
	if !s.eof {
		if slices.Contains(lines, s.input) {
			s.lastMatch = s.input
			s.lastLine = s.line
			return nil
		}
	}
	err := &UnexpectedInputError{
		EOF:      s.eof,
		Line:     s.line,
		Input:    s.input,
		Literals: lines,
	}
	s.errs.Append(err)
	return err
}

// ExpectNextRE will return nil if one of the supplied reqular expressions
// matches the next line read from the input stream. It will block waiting
// for the next line; the supplied context can be used to provide a timeout.
func (s *Lines) ExpectNextRE(ctx context.Context, expressions ...*regexp.Regexp) error {
	if err := s.nextLine(ctx); err != nil {
		err.(*UnexpectedInputError).Expressions = expressions
		s.errs.Append(err)
		return err
	}
	if !s.eof {
		for _, re := range expressions {
			if re.MatchString(s.input) {
				s.lastMatch = s.input
				s.lastLine = s.line
				return nil
			}
		}
	}
	err := &UnexpectedInputError{
		EOF:         s.eof,
		Line:        s.line,
		Input:       s.input,
		Expressions: expressions,
	}
	s.errs.Append(err)
	return err
}

// ExpectEOF will return nil if the underlying input stream is closed. It will
// block waiting for EOF; the supplied context can be used to provide a timeout.
func (s *Lines) ExpectEOF(ctx context.Context) error {
	if err := s.nextLine(ctx); err != nil {
		err.(*UnexpectedInputError).EOFExpected = true
		s.errs.Append(err)
		return err
	}
	if s.eof {
		return nil
	}
	err := &UnexpectedInputError{
		EOFExpected: true,
		Line:        s.line,
		Input:       s.input,
	}
	s.errs.Append(err)
	return err
}

func (s *Lines) eventually(ctx context.Context, literals []string, expressions []*regexp.Regexp) error {
	var err error
	for {
		select {
		case rec := <-s.ch:
			switch {
			case rec == nil:
				s.eof = true
				goto done
			case rec.err != nil:
				err = rec.err
				goto done
			}
			s.input = rec.input
			s.line++
			if slices.Contains(literals, s.input) {
				s.lastMatch = s.input
				s.lastLine = s.line
				return nil
			}
			for _, expr := range expressions {
				if expr.MatchString(s.input) {
					s.lastMatch = s.input
					s.lastLine = s.line
					return nil
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			goto done
		}
	}
done:
	err = &UnexpectedInputError{
		Err:         err,
		EOF:         s.eof,
		Input:       s.input,
		Line:        s.line,
		Eventually:  true,
		Literals:    literals,
		Expressions: expressions,
	}
	s.errs.Append(err)
	return err
}

// ExpectEventually will return nil if (and as soon as) one of the supplied lines
// equals one of the lines read from the input stream. It will block waiting for
// matching lines; the supplied context can be used to provide a timeout.
func (s *Lines) ExpectEventually(ctx context.Context, lines ...string) error {
	return s.eventually(ctx, lines, nil)
}

// ExpectEventuallyRE will return nil if (and as soon as) one of the supplied
// regular expressions matches one of the lines read from the input stream. It
// will block waiting for matching lines; the supplied context can be used to
// provide a timeout.
func (s *Lines) ExpectEventuallyRE(ctx context.Context, expressions ...*regexp.Regexp) error {
	return s.eventually(ctx, nil, expressions)
}

// LastMatch returns the line number and contents of the last successfully
// matched input line.
func (s *Lines) LastMatch() (int, string) {
	return s.lastLine, s.lastMatch
}
