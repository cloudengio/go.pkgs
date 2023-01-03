# Package [cloudeng.io/text/textutil](https://pkg.go.dev/cloudeng.io/text/textutil?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/text/textutil)](https://goreportcard.com/report/cloudeng.io/text/textutil)

```go
import cloudeng.io/text/textutil
```

Package textutil provides utility routines for working with text,
in particular utf8 encoded text.

## Functions
### Func BytesToString
```go
func BytesToString(b []byte) string
```
BytesToString returns a string with the supplied byte slice as its
contents. The original byte slice must never be modified. Taken from
strings.Builder.String().

### Func ReverseBytes
```go
func ReverseBytes(input string) []byte
```
ReverseBytes returns a new slice containing the runes in the input string in
reverse order.

### Func ReverseString
```go
func ReverseString(input string) string
```
ReverseString is like ReverseBytes but returns a string.

### Func StringToBytes
```go
func StringToBytes(s string) []byte
```
StringToBytes returns the byte slice containing the data for the
supplied string without any allocations or copies. It should only
be used when the resulting byte slice will never be modified. See
https://groups.google.com/g/golang-nuts/c/Zsfk-VMd_fU/m/O1ru4fO-BgAJ




