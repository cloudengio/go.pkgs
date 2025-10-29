# Package [cloudeng.io/webapp/webauth/acme/pebble](https://pkg.go.dev/cloudeng.io/webapp/webauth/acme/pebble?tab=doc)

```go
import cloudeng.io/webapp/webauth/acme/pebble
```


## Types
### Type Config
```go
type Config struct {
	Address           string
	ManagementAddress string
	HTTPPort          int
	TLSPort           int
	Certificate       []byte
	CertificateFile   string
	CAFile            string
	TestCertBase      string
	RootCertURL       string
	// contains filtered or unexported fields
}
```
Config represents the configuration for a pebble instance that's relevant to
using it for testing clients.

### Functions

```go
func NewConfig() Config
```
NewConfig creates a new Config instance with default values.



### Methods

```go
func (pc Config) CA() *x509.CertPool
```
CA returns a CertPool containing the root pebble CA certificate. Use it when
configuring clients to connect to the pebble instance.


```go
func (pc *Config) CreateCertsAndUpdateConfig(ctx context.Context, outputDir string) (string, error)
```
CreateCertsAndUpdateConfig uses minica to create a self-signed certificate
for use with the pebble instance. The generated certificate and key
are placed in outputDir. It returns the path to the possibly unpdated
configuration file to be used when starting pebble. Use minica to create a
self-signed certificate for the domain as per:

    	  minica -ca-cert pebble.minica.pem \
              -ca-key pebble.minica.key.pem \
              -domains localhost,pebble \
              -ip-addresses 127.0.0.1


```go
func (pc Config) GetIssuingCA(ctx context.Context) (*x509.CertPool, error)
```
IssuingCA returns a CertPool containing the issuing CA certificate used by
pebble to sign issued certificates.


```go
func (pc Config) GetIssuingCert(ctx context.Context) ([]byte, error)
```
GetIssuingCert retrieves the pebble certificate, including intermediates,
used to sign issued certificates.




### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T manages a pebble instance for testing purposes.

### Functions

```go
func New(binary string) *T
```
New creates a new Pebble instance. The supplied configFile will be used to
configure the pebble instance. The server is not started by New.



### Methods

```go
func (p *T) EnsureStopped(ctx context.Context, waitFor time.Duration) error
```
EnsureStopped ensures that the pebble instance is stopped.


```go
func (p *T) PID() int
```
PID returns the process ID of the pebble instance.


```go
func (p *T) Start(ctx context.Context, dir, cfg string, forward io.WriteCloser) error
```
Start the pebble instance with its output forwarded to the supplied writer.


```go
func (p *T) WaitForIssuedCertificateSerial(ctx context.Context) (string, error)
```
WaitForIssuedCertificateSerial waits until a certificate is issued and
returns its serial number.


```go
func (p *T) WaitForReady(ctx context.Context) error
```







