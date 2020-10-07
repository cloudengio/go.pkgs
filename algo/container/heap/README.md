# Package [cloudeng.io/algo/container/heap](https://pkg.go.dev/cloudeng.io/algo/container/heap?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/algo/container/heap)](https://goreportcard.com/report/cloudeng.io/algo/container/heap)

```go
import cloudeng.io/algo/container/heap
```

Package heap contains various implementations of heap containers.

## Types
### Type KeyedInt64
```go
type KeyedInt64 struct {
	// contains filtered or unexported fields
}
```
KeyedInt64 implements a heap whose values include both a key and value to
allow for updates to existing items in the heap. It also keeps a running sum
of the all of the items currently in the heap, supports both ascending and
desencding operations. It is safe for concurrent use.

### Functions

```go
func NewKeyedInt64(descending bool) *KeyedInt64
```
NewKeyedInt64



### Methods

```go
func (ki *KeyedInt64) GobDecode(buf []byte) error
```


```go
func (ki *KeyedInt64) GobEncode() ([]byte, error)
```


```go
func (ki *KeyedInt64) Len() int
```


```go
func (ki *KeyedInt64) MarshalJSON() ([]byte, error)
```


```go
func (ki *KeyedInt64) Pop() (string, int64)
```


```go
func (ki *KeyedInt64) Remove(key string)
```


```go
func (ki *KeyedInt64) TopN(n int) []struct {
	Key   string
	Value int64
}
```


```go
func (ki *KeyedInt64) Total() int64
```


```go
func (ki *KeyedInt64) UnmarshalJSON(buf []byte) error
```


```go
func (ki *KeyedInt64) Update(key string, value int64)
```







