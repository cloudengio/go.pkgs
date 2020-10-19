# Package [cloudeng.io/cmdutil/profiling](https://pkg.go.dev/cloudeng.io/cmdutil/profiling?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/profiling)](https://goreportcard.com/report/cloudeng.io/cmdutil/profiling)

```go
import cloudeng.io/cmdutil/profiling
```

Package profiling provides stylised support for enabling profiling of
command line tools.

## Functions
### Func EnableCPUProfiling
```go
func EnableCPUProfiling(filename string) (func() error, error)
```

### Func StartProfile
```go
func StartProfile(name, filename string) (func() error, error)
```



## Types
### Type ProfileFlag
```go
type ProfileFlag struct {
	Profiles []ProfileSpec
}
```

### Methods

```go
func (pf *ProfileFlag) Get() interface{}
```


```go
func (pf *ProfileFlag) Set(v string) error
```


```go
func (pf *ProfileFlag) String() string
```




### Type ProfileSpec
```go
type ProfileSpec struct {
	Name     string
	Filename string
}
```





