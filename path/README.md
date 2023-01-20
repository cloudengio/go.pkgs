# Package [cloudeng.io/path](https://pkg.go.dev/cloudeng.io/path?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/path)](https://goreportcard.com/report/cloudeng.io/path)

```go
import cloudeng.io/path
```


## Types
### Type Sharder
```go
type Sharder interface {
	Assign(path string) (prefix, suffix string)
}
```
Sharder is the interface for assigning and managing pathnames to shards.

### Functions

```go
func NewSharder(opts ...ShardingOption) Sharder
```
NewSharder returns an instance of Sharder according to the specified
options. If no options are provided it will behave as if the option of
WithSHA1PrefixLength(2) was used.




### Type ShardingOption
```go
type ShardingOption func(o *shardingOptions)
```
ShardingOption represents an option to NewPathSharder.

### Functions

```go
func WithSHA1PrefixLength(v int) ShardingOption
```
WithSHA1PrefixLength requests that a SHA1 sharder with a prefix
length of v is used. Assigned filenames will be of the form:
sha1(path)[:v]/sha1(path)[v:]







