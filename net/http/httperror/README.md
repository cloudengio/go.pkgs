# Package [cloudeng.io/net/http/httperror](https://pkg.go.dev/cloudeng.io/net/http/httperror?tab=doc)

```go
import cloudeng.io/net/http/httperror
```


## Functions
### Func CheckResponse
```go
func CheckResponse(err error, resp *http.Response) error
```
CheckResponse creates a new instance of T given an error and http.Response
returned by an http request operation (e.g. Get, Do etc). If err is nil,
resp must not be nil. It will return nil if err is nil and resp.StatusCode
is http.StatusOK. Otherwise, it will create an instance of httperror.T with
the appropriate fields set.

### Func CheckResponseRetries
```go
func CheckResponseRetries(err error, resp *http.Response, retries int) error
```
CheckResponseRetries is like Checkresponse but will set the retries field.

### Func IsHTTPError
```go
func IsHTTPError(err error, httpStatusCode int) bool
```
IsHTTPError returns true if err contains the specified http status code.



## Types
### Type T
```go
type T struct {
	Err        error
	Status     string
	StatusCode int
	Retries    int
}
```
T represents an error encountered while making an HTTP request of some form.
The error may be the result of a failed local operation (in which case Err
will be non-nil), or an error returned by the remote server (in which case
Err will be nil but StatusCode will something other than http.StatusOK).
In all cases, Err or StatusCode must contain an error.

### Functions

```go
func AsT(httpStatusCode int) *T
```
AsT returns an httperror.T for the specified http status code.



### Methods

```go
func (err *T) Error() string
```
Error implements error.


```go
func (err *T) Is(target error) bool
```
Is implements errors.Is.







