# Package [cloudeng.io/logging](https://pkg.go.dev/cloudeng.io/logging?tab=doc)

```go
import cloudeng.io/logging
```


## Types
### Type JSONFormatter
```go
type JSONFormatter struct {
	// contains filtered or unexported fields
}
```
JSONFormatter implements a log formatter that outputs JSON formatted logs.

### Functions

```go
func NewJSONFormatter(w io.Writer, prefix, indent string) *JSONFormatter
```
NewJSONFormatter creates a new JSONFormatter that writes to with the
specified prefix and indent.



### Methods

```go
func (js *JSONFormatter) Format(v any) error
```
Format formats the specified value as JSON.


```go
func (js *JSONFormatter) Write(p []byte) (n int, err error)
```
Write implements the io.Writer interface and it assumes that it is called
with a complete JSON object.







