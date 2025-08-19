# Package [cloudeng.io/webapp/cookies](https://pkg.go.dev/cloudeng.io/webapp/cookies?tab=doc)

```go
import cloudeng.io/webapp/cookies
```


## Types
### Type ScopeAndDuration
```go
type ScopeAndDuration struct {
	Domain   string
	Path     string
	Duration time.Duration
}
```
ScopeAndDuration represents the scope and duration settings for cookies.

### Methods

```go
func (d ScopeAndDuration) Cookie(value string) *http.Cookie
```
Cookie returns a new http.Cookie with the specified value and the scope and
duration settings from the ScopeAndDuration receiver.


```go
func (d ScopeAndDuration) SetDefaults(domain, path string, duration time.Duration) ScopeAndDuration
```
SetDefaults uses the supplied values as defaults for ScopeAndDuration if the
current values are not already set.




### Type Secure
```go
type Secure string
```
Secure represents a named cookie that is set 'securely'. It is primarily
intended to document and track the use of cookies in a web application.

### Methods

```go
func (c Secure) Read(r *http.Request) (string, bool)
```
Read reads the cookie from the request and returns its value. If the cookie
is not present, it returns an empty string and false.


```go
func (c Secure) ReadAndClear(rw http.ResponseWriter, r *http.Request) (string, bool)
```
ReadAndClear reads a cookie and requests its removal by setting its MaxAge
to -1 and its value to an empty string.


```go
func (c Secure) Set(rw http.ResponseWriter, ck *http.Cookie)
```
Set sets the supplied cookie securely with the name of the cookie specified
in the receiver and secure values for HttpOnly, Secure and SameSite (true,
true, SameSiteStrictMode). All other fields in ck are used as specified.




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
receiver. It overwrites the Name in ck. All other fields in ck are used as
specified.







