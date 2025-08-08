# Package [cloudeng.io/webapp/cookies](https://pkg.go.dev/cloudeng.io/webapp/cookies?tab=doc)

```go
import cloudeng.io/webapp/cookies
```


## Types
### Type T
```go
type T string
```
T represents a named cookie. It is primarily intended to document and track
the use of cookies in a web application.

### Methods

```go
func (c T) Read(r *http.Request) (string, bool)
```
Read reads the cookie from the request and returns its value. If the cookie
is not present, it returns an empty string and false.


```go
func (c T) ReadAndClear(rw http.ResponseWriter, r *http.Request) (string, bool)
```
ReadAndClear reads a cookie and requests its removal by setting its MaxAge
to -1 and its value to an empty string.


```go
func (c T) Set(rw http.ResponseWriter, ck *http.Cookie)
```
Set sets the supplied cookie with the name of the cookie specified in the
receiver but using all other values from the supplied cookie.


```go
func (c T) SetSecureWithExpiration(rw http.ResponseWriter, value string, expires time.Duration)
```
SetSecureWithExpiration sets a cookie with a specific expiration time.
The path is set to "/" to make the cookie available across the entire site,
and the cookie is marked as secure and HTTP-only and SameSiteStrictMode.







