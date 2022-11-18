# Package [cloudeng.io/errors](https://pkg.go.dev/cloudeng.io/errors?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/errors)](https://goreportcard.com/report/cloudeng.io/errors)

```go
import cloudeng.io/errors
```

Package errors provides utility routines for working with errors that are
compatible with go 1.13+ and for annotating errors. It provides errors.M
which can be used to collect and work with multiple errors in a thread safe
manner. It also provides convenience routines for annotating existing errors
with caller and other information.

    errs := errors.M{}
    errs.Append(fn(a))
    errs.Append(fn(b))
    err := errs.Err()

The location of a function's immediate caller (depth of 1) in form of the
directory/filename:<line> (name len of 2) can be obtained as follows:

    errors.Caller(1, 2)

Annotations, can be added as follows:

    err := errors.WithCaller(os.ErrNotExist)

Where:

    fmt.Printf("%v\n", err)
    fmt.Printf("%v\n", errors.Unwrap(err))

Would produce:

    errors/caller_test.go:17: file does not exist
    file does not exist

Annotated errors can be passed to errors.M:

    errs := errors.M{}
    errs.Append(errors.WithCaller(fn(a)))
    errs.Append(errors.WithCaller(fn(b)))
    err := errs.Err()

## Functions
### Func Annotate
```go
func Annotate(annotation string, err error) error
```
Annotate returns an error representing the original error and the supplied
annotation.

### Func AnnotateAll
```go
func AnnotateAll(annotation string, errs ...error) []error
```
AnnotateAll returns a slice of errors representing the original errors and
the supplied annotation.

### Func As
```go
func As(err error, target interface{}) bool
```
As calls errors.As.

### Func Caller
```go
func Caller(depth, nameLen int) string
```
Caller returns the caller's location as a filepath and line number. Depth
follows the convention for runtime.Caller. The filepath is the trailing
nameLen components of the filename returned by runtime.Caller. A nameLen of
2 is generally the best compromise between brevity and precision since it
includes the enclosing directory component as well as the filename.

### Func Is
```go
func Is(err, target error) bool
```
Is calls errors.Is.

### Func New
```go
func New(m string) error
```
New calls errors.New.

### Func NewM
```go
func NewM(errs ...error) error
```
NewM is equivalent to:

    errs := errors.M{}
    ...
    errs.Append(err)
    ...
    return errs.Err()

### Func Unwrap
```go
func Unwrap(err error) error
```
Unwrap calls errors.Unwrap.

### Func WithCaller
```go
func WithCaller(err error) error
```
WithCaller returns an error annotated with the location of its immediate
caller.

### Func WithCallerAll
```go
func WithCallerAll(err ...error) []error
```
WithCallerAll returns a slice conntaing annotated versions of all of the
supplied errors.



## Types
### Type M
```go
type M struct {
	// contains filtered or unexported fields
}
```
M represents multiple errors. It is thread safe. Typical usage is:

    errs := errors.M{}
    ...
    errs.Append(err)
    ...
    return errs.Err()

### Methods

```go
func (m *M) Append(errs ...error)
```
Append appends the specified errors excluding nil values.


```go
func (m *M) As(target interface{}) bool
```
As supports errors.As.


```go
func (m *M) Clone() *M
```
Clone returns a new errors.M that contains the same errors as itself.


```go
func (m *M) Err() error
```
Err returns nil if m contains no errors, or itself otherwise.


```go
func (m *M) Error() string
```
Error implements error.error


```go
func (m *M) Format(f fmt.State, c rune)
```
Format implements fmt.Formatter.Format.


```go
func (m *M) Is(target error) bool
```
Is supports errors.Is.


```go
func (m *M) Unwrap() error
```
Unwrap implements errors.Unwrap. It returns the first stored error and then
removes that error.






## Examples
### [ExampleCaller](https://pkg.go.dev/cloudeng.io/errors?tab=doc#example-Caller)

### [ExampleWithCaller](https://pkg.go.dev/cloudeng.io/errors?tab=doc#example-WithCaller)

### [ExampleM](https://pkg.go.dev/cloudeng.io/errors?tab=doc#example-M)

### [ExampleM_caller](https://pkg.go.dev/cloudeng.io/errors?tab=doc#example-M_caller)




