# errors 

[![CircleCI](https://circleci.com/gh/cloudengio/go.pkgs.svg?style=svg)](https://circleci.com/gh/cloudengio/go.pkgs)

errors provides utility routines for working with errors that are compatible with go 1.13+
and for annotating errors with location and other information.

It currently provides:

1. `errors.M` which can be used to store multiple error values. `errors.M` is thread safe.

```go
errs := errors.M{}
...
errs.Append(fn(a))
...
errs.Append(fn(b))
...
err := errs.Err()
```

2. `errors.Caller`, `errors.Annotate` and `errors.AnnotateAll` which can be used to annotate an existing
error with location information on the caller or an arbitrary string respectively.

```go
err := errors.Caller(os.ErrNotExist)
fmt.Printf("%v\n", err)
fmt.Printf("%v\n", errors.Unwrap(err))
// Output:
// errors/caller_test.go:17: file does not exist
// file does not exist
```
