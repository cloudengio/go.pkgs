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

### Functions

```go
func NewJSONFormatter(w io.Writer, prefix, indent string) *JSONFormatter
```



### Methods

```go
func (js *JSONFormatter) Format(v any) error
```


```go
func (js *JSONFormatter) Write(p []byte) (n int, err error)
```







