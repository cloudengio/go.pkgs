# Package [cloudeng.io/cmdutil/profiling](https://pkg.go.dev/cloudeng.io/cmdutil/profiling?tab=doc)

```go
import cloudeng.io/cmdutil/profiling
```

Package profiling provides support for enabling profiling of command line
tools via flags.

## Functions
### Func IsPredefined
```go
func IsPredefined(name string) bool
```
IsPredefined returns true if the specified name is one of the pprof
predefined profiles, or 'cpu' which is recognised by this package as
requesting a cpu profile.

### Func PredefinedProfiles
```go
func PredefinedProfiles() []string
```
PredefinedProfiles returns the list of predefined profiles, ie.
those documented as 'predefined' by the runtime/pprof package, such as
"goroutine", "heap", "allocs", "threadcreate", "block", "mutex".

### Func Start
```go
func Start(name, filename string) (func() error, error)
```
Start enables the named profile and returns a function that can be used to
save its contents to the specified file. Typical usage is as follows:

    save, err := profiling.Start("cpu", "cpu.out")
    if err != nil {
       panic(err)
    }
    defer save()

For a heap profile simply use Start("heap", "heap.out"). Note that the
returned save function cannot be used more than once and that Start must be
called multiple times to create multiple heap output files for example. All
of the predefined named profiles from runtime/pprof are supported. If a new,
custom profile is requested, then the caller must obtain a reference to it
via pprof.Lookup and the create profiling records appropriately.

### Func StartFromSpecs
```go
func StartFromSpecs(specs ...ProfileSpec) (func(), error)
```
StartFromSpecs starts all of the specified profiles.



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





