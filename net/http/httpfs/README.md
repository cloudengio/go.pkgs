# Package [cloudeng.io/net/http/httpfs](https://pkg.go.dev/cloudeng.io/net/http/httpfs?tab=doc)

```go
import cloudeng.io/net/http/httpfs
```


## Functions
### Func New
```go
func New(client *http.Client, options ...Option) file.FS
```
New creates a new instance of file.FS backed by http/https.



## Types
### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithHTTPScheme() Option
```




### Type Response
```go
type Response struct {
	// When the response was received.
	When time.Time

	// Fields copied from the http.Response.
	Headers                http.Header
	Trailers               http.Header
	ContentLength          int64
	StatusCode             int
	ProtoMajor, ProtoMinir int
	TransferEncoding       []string
}
```
Response is a redacted version of http.Response that can be marshaled using
gob.





