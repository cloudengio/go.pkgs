# Package [cloudeng.io/cmdutil/profiling](https://pkg.go.dev/cloudeng.io/cmdutil/profiling?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/profiling)](https://goreportcard.com/report/cloudeng.io/cmdutil/profiling)

```go
import cloudeng.io/cmdutil/profiling
```

Package profiling provides support for enabling profiling of command line
tools via flags.

## Functions
### Func Start
```go
func Start(name, filename string) (func() error, error)
```
Start enables the named profile and returns a function that can be used to
save its contents to the specified file. Typical usage is as follows:

save, err := profiling.Start("cpu", "cpu.out") if err != nil {

    panic(err)

} defer save()

For a heap profile simply use Start("heap", "heap.out"). Note that the
returned save function cannot be used more than once and that Start must be
called multiple times to create multiple heap output files for example. All
of the predefined named profiles from runtime/pprof are supported. If a new,
custom profile is requested, then the caller must obtain a reference to it
via pprof.Lookup and the create profiling records appropriately.



## Types
### Type ProfileFlag
```go
type ProfileFlag struct {
	Profiles []ProfileSpec
}
```
ProfileFlag can be used to represent flags to request arbritrary profiles.

### Methods

```go
func (pf *ProfileFlag) Get() interface{}
```
Get implements flag.Getter.


```go
func (pf *ProfileFlag) Set(v string) error
```
Set implements flag.Value.


```go
func (pf *ProfileFlag) String() string
```
String implements flag.Value.




### Type ProfileSpec
```go
type ProfileSpec struct {
	Name     string
	Filename string
}
```
ProfileSpec represents a named profile and the name of the file to write its
contents to. CPU profiling can be requested using the name 'cpu' rather than
the CPUProfiling API calls in runtime/pprof that predate the named profiles.





