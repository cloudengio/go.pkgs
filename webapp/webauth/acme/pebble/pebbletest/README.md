# Package [cloudeng.io/webapp/webauth/acme/pebble/pebbletest](https://pkg.go.dev/cloudeng.io/webapp/webauth/acme/pebble/pebbletest?tab=doc)

```go
import cloudeng.io/webapp/webauth/acme/pebble/pebbletest
```


## Functions
### Func WaitForNewCert
```go
func WaitForNewCert(ctx context.Context, t Testing, msg, certPath string, previousSerial string) (*x509.Certificate, *x509.CertPool)
```
WaitForNewCert waits for a new certificate to be issued at certPath with a
serial number different from previousSerial.



## Types
### Type Recorder
```go
type Recorder struct {
	// contains filtered or unexported fields
}
```
Recorder is an io.WriteCloser that records all data written to it.

### Functions

```go
func Start(ctx context.Context, t Testing, tmpDir string, configOpts ...pebble.ConfigOption) (*pebble.T, pebble.Config, *Recorder, string, string)
```
Start starts a pebble ACME server for testing purposes.



### Methods

```go
func (o *Recorder) Close() error
```


```go
func (o *Recorder) String() string
```


```go
func (o *Recorder) Write(p []byte) (n int, err error)
```




### Type Testing
```go
type Testing interface {
	Fatalf(format string, args ...any)
	Helper()
	Logf(format string, args ...any)
}
```





