# Package [cloudeng.io/webapp/jsonapi](https://pkg.go.dev/cloudeng.io/webapp/jsonapi?tab=doc)

```go
import cloudeng.io/webapp/jsonapi
```

Package jsonapi provides utilities for working with json REST APIs.

## Functions
### Func WriteError
```go
func WriteError(rw http.ResponseWriter, err ErrorResponse, status int)
```
WriteError writes an ErrorResponse in JSON format to the
http.ResponseWriter. It sets the appropriate HTTP status code and content
type.

### Func WriteErrorMsg
```go
func WriteErrorMsg(rw http.ResponseWriter, msg string, status int)
```
WriteErrorMsg writes an error message in JSON format to the
http.ResponseWriter using WriteErrror.



## Types
### Type Endpoint
```go
type Endpoint[Req, Resp any] struct{}
```
Endpoint represents a JSON API endpoint with a request and response type.
It provides methods to parse the request from an io.Reader and write the
response to an io.Writer. If an error occurs during parsing or writing,
it can write an error response in JSON format using the WriteError method.
It is primarily intended to identify and document JSON API endpoints.

### Methods

```go
func (ep Endpoint[Req, Resp]) ParseRequest(rw http.ResponseWriter, r *http.Request, req *Req) error
```
ParseRequest reads the request body from the provided http.Request and
decodes it into the Request field of the Endpoint. If decoding failes,
it uses WriteError to write an error message to the client.


```go
func (ep Endpoint[Req, Resp]) WriteResponse(rw http.ResponseWriter, resp Resp) error
```
WriteResponse writes the response in JSON format to the http.ResponseWriter.
It sets the Content-Type header to "application/json" and writes the HTTP
status code. If encoding the response fails, it uses WriteError to write an
error message to the client.




### Type ErrorResponse
```go
type ErrorResponse struct {
	Message string `json:"message"`
}
```
ErrorResponse represents a JSON error response.





