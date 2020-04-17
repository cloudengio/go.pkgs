# Package [cloudeng.io/text/edit](https://pkg.go.dev/cloudeng.io/text/edit?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/text/edit)](https://goreportcard.com/report/cloudeng.io/text/edit)

```go
import cloudeng.io/text/edit
```

Package edit provides support for editing in-memory byte slices using
insert, delete and replace operations.

## Functions
### Func Do
```go
func Do(contents []byte, deltas ...Delta) []byte
```
Do applies the supplied deltas to contents as follows:

    1. Deltas are sorted by their start position, then at each position,
    2. deletions are applied, then
    3. replacements are applied, then,
    4. insertions are applied.

Sorting is stable with respect the order specified in the function
invocation. Multiple deletions and replacements overwrite each other,
whereas insertions are concatenated. All position values are with respect to
the original value of contents.

### Func DoString
```go
func DoString(contents string, deltas ...Delta) string
```
DoString is like Do but for strings.

### Func Validate
```go
func Validate(contents []byte, deltas ...Delta) error
```
Validate determines if the supplied deltas fall within the bounds of
content.



## Types
### Type Delta
```go
type Delta struct {
	// contains filtered or unexported fields
}
```
Delta represents an insertion, deletion or replacement.



## Examples

### [ExampleDo](https://pkg.go.dev/cloudeng.io/text/edit?tab=doc#example-Do)



