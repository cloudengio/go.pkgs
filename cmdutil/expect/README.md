# Package [cloudeng.io/cmdutil/expect](https://pkg.go.dev/cloudeng.io/cmdutil/expect?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/expect)](https://goreportcard.com/report/cloudeng.io/cmdutil/expect)

```go
import cloudeng.io/cmdutil/expect
```

Package expect provides support for making expectations on the contents of
input streams.

## Types
### Type Lines
```go
type Lines struct {
	// contains filtered or unexported fields
}
```
Lines provides line oriented expecations and will block waiting for the
expected input. A context with a timeout or deadline can be used to abort
the expectation. Literal and regular expression matches are supported as is
matching on EOF. Each operation accepts multiple literals or regular
expressions that are treated as an 'or' to allow for convenient handling of
different input orderings.

### Functions

```go
func NewLineStream(rd io.Reader, opts ...Option) *Lines
```
NewLineStream creates a new instance of Lines.



### Methods

```go
func (s *Lines) Err() error
```
Err returns all errors encountered. Note that closing the underlying
io.Reader is not considered an error unless ExpectEOF failed.


```go
func (s *Lines) ExpectEOF(ctx context.Context) error
```
ExpectEOF will return nil if the underlying input stream is closed. It will
block waiting for EOF; the supplied context can be used to provide a
timeout.


```go
func (s *Lines) ExpectEventually(ctx context.Context, lines ...string) error
```
ExpectEventually will return nil if (and as soon as) one of the supplied
lines equals one of the lines read from the input stream. It will block
waiting for matching lines; the supplied context can be used to provide a
timeout.


```go
func (s *Lines) ExpectEventuallyRE(ctx context.Context, expressions ...*regexp.Regexp) error
```
ExpectEventuallyRE will return nil if (and as soon as) one of the supplied
regular expressions matches one of the lines read from the input stream. It
will block waiting for matching lines; the supplied context can be used to
provide a timeout.


```go
func (s *Lines) ExpectNext(ctx context.Context, lines ...string) error
```
ExpectNext will return nil if one of the supplied lines is equal to the next
line read from the input stream. It will block waiting for the next line;
the supplied context can be used to provide a timeout.


```go
func (s *Lines) ExpectNextRE(ctx context.Context, expressions ...*regexp.Regexp) error
```
ExpectNextRE will return nil if one of the supplied reqular expressions
matches the next line read from the input stream. It will block waiting for
the next line; the supplied context can be used to provide a timeout.


```go
func (s *Lines) LastMatch() (int, string)
```
LastMatch returns the line number and contents of the last successfully
matched input line.




### Type Option
```go
type Option func(*options)
```
Option represents an option.

### Functions

```go
func TraceInput(out io.Writer) Option
```
TraceInput enables tracing of input as it is read.




### Type UnexpectedInputError
```go
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
```
UnexpectedInputError represents a failed expectation, i.e. when the contents
of the input do not match the expected contents.

### Methods

```go
func (e *UnexpectedInputError) Error() string
```
Error implements error.


```go
func (e *UnexpectedInputError) Expectation() string
```






## Examples
### [ExampleLines](https://pkg.go.dev/cloudeng.io/cmdutil/expect?tab=doc#example-Lines)




