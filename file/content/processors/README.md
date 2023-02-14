# Package [cloudeng.io/file/content/processors](https://pkg.go.dev/cloudeng.io/file/content/processors?tab=doc)

```go
import cloudeng.io/file/content/processors
```

Package processor provides support for processing different content types.

## Types
### Type HTML
```go
type HTML struct{}
```
HTML provides support for processing HTML documents.

### Methods

```go
func (ho HTML) Parse(rd io.Reader) (HTMLDoc, error)
```




### Type HTMLDoc
```go
type HTMLDoc struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (ho HTMLDoc) HREFs(base string) ([]string, error)
```
HREFs returns the hrefs found in the provided HTML document.


```go
func (ho HTMLDoc) Title() string
```







