# Package [cloudeng.io/debug/goroutines](https://pkg.go.dev/cloudeng.io/debug/goroutines?tab=doc)

```go
import cloudeng.io/debug/goroutines
```


## Functions
### Func CompactTemplate
```go
func CompactTemplate() (*template.Template, error)
```
CompactTemplate returns a single-line-per-frame representation template that
emits concise goroutine information.

### Func Format
```go
func Format(gs ...*Goroutine) string
```
Format formats Goroutines back into the normal string representation.

### Func FormatWithTemplate
```go
func FormatWithTemplate(tmpl *template.Template, gs ...*Goroutine) (string, error)
```
FormatWithTemplate renders a collection of goroutines using the supplied
template. The template is executed once for each line that would appear in
the textual stack trace: the goroutine header, each frame, and the optional
creator frame. The provided TemplateData exposes the raw goroutine/frame
values along with helper booleans that enable conditional formatting from
the template itself.

### Func PanicTemplate
```go
func PanicTemplate() (*template.Template, error)
```
PanicTemplate returns a template that mimics the formatting produced by a Go
panic stack trace. The returned template is a clone of an internal instance,
so callers may modify it without affecting future calls.



## Types
### Type Frame
```go
type Frame struct {
	Call   string
	File   string
	Line   int64
	Offset int64
}
```
Frame represents a single stack frame.


### Type Goroutine
```go
type Goroutine struct {
	ID      int64
	State   string
	Stack   []*Frame
	Creator *Frame
}
```
Goroutine represents a single goroutine.

### Functions

```go
func Get(ignore ...string) ([]*Goroutine, error)
```
Get gets a set of currently running goroutines and parses them into a
structured representation. Any goroutines that match the ignore list are
ignored.


```go
func Parse(buf []byte, ignore ...string) ([]*Goroutine, error)
```
Parse parses a stack trace into a structure representation.




### Type TemplateData
```go
type TemplateData struct {
	Goroutine *Goroutine
	Frame     *Frame

	GoroutineIndex int
	GoroutineCount int
	FrameIndex     int
	FrameCount     int

	IsHeader         bool
	IsFrame          bool
	IsCreator        bool
	IsFirstGoroutine bool
	IsLastGoroutine  bool
	IsFirstFrame     bool
	IsLastFrame      bool
	HasFrames        bool
	HasCreator       bool
	HasOffset        bool
	OffsetHex        string
}
```
TemplateData provides the context passed to templates executed by
FormatWithTemplate. Fields are exported so they can be accessed from the
template, including convenience booleans describing the position of the
current line.





