# Package [cloudeng.io/file/content/processors](https://pkg.go.dev/cloudeng.io/file/content/processors?tab=doc)

```go
import cloudeng.io/file/content/processors
```

Package processor provides support for processing different content types.

## Types
### Type Email
```go
type Email struct{}
```


### Type EmailDoc
```go
type EmailDoc struct {
	From    string          // Email address of the sender
	To      []*mail.Address // List of recipient email addresses
	Subject string          // Subject of the email
	Date    string          // Date the email was sent
	Body    []byte          // Body of the email
	Raw     []byte          // Raw email content
	Labels  []string        // Labels or tags associated with the email
}
```

### Methods

```go
func (EmailDoc) Parse(rawEmail []byte) (EmailDoc, error)
```




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







