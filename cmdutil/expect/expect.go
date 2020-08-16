package expect

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"cloudeng.io/errors"
)

// UnexpectedInputError represents a failed expectation, i.e. when
// the input in the stream does not match that requested.
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
		fmt.Fprintf(out, ":%s", e.Literals[0])
	default:
		fmt.Fprintf(out, ":\n")
		for _, l := range e.Literals {
			fmt.Fprintf(out, "\t%s\n", l)
		}
	}
	return
}

func (e *UnexpectedInputError) formatREs(out io.Writer) {
	switch len(e.Expressions) {
	case 0:
	case 1:
		fmt.Fprintf(out, ": %s", e.Expressions[0])
	default:
		fmt.Fprintf(out, ":\n")
		for _, re := range e.Expressions {
			fmt.Fprintf(out, "\t%s\n", re)
		}
	}
	return
}

func (e *UnexpectedInputError) Error() string {
	opname := e.Expectation()
	out := &strings.Builder{}
	fmt.Fprintf(out, "%s: failed @ %v:(%v)", opname, e.Line, e.Input)
	e.formatLiterals(out)
	e.formatREs(out)
	return out.String()
}

type Stream struct {
	rd    io.Reader
	brd   *bufio.Reader
	eof   bool
	input string
	line  int
	errs  *errors.M
}

func New(rd io.Reader) *Stream {
	s := &Stream{
		rd:   rd,
		errs: &errors.M{},
	}
	s.initReader()
	return s
}

type record struct {
	input string
	err   error
}

func (s *Stream) copy(ctx context.Context, ch chan<- error) {

	for {
		str, err := s.brd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.eof = true
				err = nil
			}
			ch <- err
			return
		}
		s.input = strings.TrimSuffix(str, "\n")
		s.line++
		ch <- nil
	}
}

func (s *Stream) initReader() {
	s.prd, s.prw = io.Pipe()
	s.brd = bufio.NewReader(s.prd)
	go func() {
		for {
			n, err := io.Copy(s.prw, s.rd)
			fmt.Printf("%v .. %v\n", n, err)
		}
	}()
}

// Err returns all errors encountered. Note that ending of the stream
// is not considered an error unless ExpectEOF failed.
func (s *Stream) Err() error {
	return s.errs.Err()
}

func (s *Stream) nextLine(ctx context.Context) error {
	ch := make(chan error, 1)
	fmt.Printf("new chanel %v\n", ch)
	go func() {
		fmt.Printf("calling scan\n")
		str, err := s.brd.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				s.eof = true
				err = nil
			}
			ch <- err
			return
		}
		s.input = strings.TrimSuffix(str, "\n")
		s.line++
		ch <- nil
	}()
	fmt.Printf("cvalling select: %v\n", ch)
	select {
	case err := <-ch:
		fmt.Printf("select %v: got input: %v\n", ch, err)
		if err != nil {
			return &UnexpectedInputError{
				Err:   err,
				Line:  s.line,
				EOF:   s.eof,
				Input: s.input,
			}
		}
	case <-ctx.Done():
		fmt.Printf("select %v: canceled %v\n", ch, ctx.Err())
		return &UnexpectedInputError{
			Err:   ctx.Err(),
			EOF:   s.eof,
			Line:  s.line,
			Input: s.input,
		}
	}
	return nil
}

func (s *Stream) ExpectNext(ctx context.Context, lines ...string) error {
	if err := s.nextLine(ctx); err != nil {
		s.errs.Append(err)
		return err
	}
	if !s.eof {
		for _, line := range lines {
			if line == s.input {
				return nil
			}
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

func (s *Stream) ExpectNextRE(ctx context.Context, expressions ...*regexp.Regexp) error {
	if err := s.nextLine(ctx); err != nil {
		s.errs.Append(err)
		return err
	}
	if !s.eof {
		for _, re := range expressions {
			if re.MatchString(s.input) {
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

func (s *Stream) ExpectEOF(ctx context.Context) error {
	if err := s.nextLine(ctx); err != nil {
		fmt.Printf("nextLine: %v\n", err)
		err.(*UnexpectedInputError).EOFExpected = true
		s.errs.Append(err)
		return err
	}
	fmt.Printf("nextLine: %v\n", s)

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

func (s *Stream) eventually(ctx context.Context, literals []string, expressions []*regexp.Regexp) error {
	type record struct {
		found bool
		err   error
	}
	resultCh := make(chan record, 1)
	go func() {
		for {
			str, err := s.brd.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					s.eof = true
				}
				resultCh <- record{err: err}
				return
			}
			s.input = strings.TrimSuffix(str, "\n")
			s.line++
			for _, line := range literals {
				if s.input == line {
					resultCh <- record{found: true}
					return
				}
			}
			for _, expr := range expressions {
				if expr.MatchString(s.input) {
					resultCh <- record{found: true}
					return
				}
			}
		}
	}()
	var err error
	select {
	case rec := <-resultCh:
		if rec.found {
			return nil
		}
	case <-ctx.Done():
		err = ctx.Err()
	}
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

func (s *Stream) ExpectEventually(ctx context.Context, lines ...string) error {
	return s.eventually(ctx, lines, nil)
}

func (s *Stream) ExpectEventuallyRE(ctx context.Context, expressions ...*regexp.Regexp) error {
	return s.eventually(ctx, nil, expressions)
}

// TODO: add expectVAR - or rather regexpt for <NAME>=<XXX>
