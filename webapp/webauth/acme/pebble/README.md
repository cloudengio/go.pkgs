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
func NewConfig(opt ...ConfigOption) Config
```
NewConfig creates a new Config instance with default values.



### Methods

```go
func (pc Config) CA() *x509.CertPool
```
CA returns a CertPool containing the root pebble CA certificate. Use it when
configuring clients to connect to the pebble instance.


```go
func (pc Config) CARootsURL(id int) string
```
CARootsURL returns the URL from which the pebble root CA certificate can be
retrieved, use 0 as the id.


```go
func (pc *Config) CreateCertsAndUpdateConfig(ctx context.Context, outputDir string) (string, error)
```
CreateCertsAndUpdateConfig uses minica to create a self-signed certificate
for use with the pebble instance and applies any other config customizations
requested by any ConfigOptions. The generated certificate and key are placed
in outputDir. It returns the path to the possibly updated configuration
file to be used when starting pebble. Use minica to create a self-signed
certificate for the domain as per:

    	  minica -ca-cert pebble.minica.pem \
              -ca-key pebble.minica.key.pem \
              -domains localhost,pebble \
              -ip-addresses 127.0.0.1


```go
func (pc Config) DirectoryURL() string
```
DirectoryURL returns the ACME service 'directory' URL.


```go
func (pc Config) GetIssuingCA(ctx context.Context, id int) (*x509.CertPool, error)
```
IssuingCA returns a CertPool containing the issuing CA certificate used by
pebble to sign issued certificates.


```go
func (pc Config) GetIssuingCert(ctx context.Context, id int) ([]byte, error)
```
GetIssuingCert retrieves the pebble certificate, including intermediates,
used to sign issued certificates.


```go
func (pc Config) PossibleValidityPeriods() []time.Duration
```
PossibleValidityPeriods returns the validity periods specified across all
defined profiles in the pebble config.


```go
func (pc Config) ValidateCertificate(ctx context.Context, cert *x509.Certificate, intermediates *x509.CertPool) error
```




### Type ConfigOption
```go
type ConfigOption func(*configOption)
```
ConfigOption represents an option for configuring a new Config instance.

### Functions

```go
func WithValidityPeriod(secs int) ConfigOption
```
WithValidityPeriod returns a ConfigOption that sets the validity period for
all issued certificates by modifying the value in all pebble profiles.




### Type ServerOption
```go
type ServerOption func(*serverOptions)
```
ServerOption represents a option for configuring a new Pebble instance.


### Type T
```go
type T struct {
	// contains filtered or unexported fields
}
```
T manages a pebble instance for testing purposes.

### Functions

```go
func New(binary string, opts ...ServerOption) *T
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
func (p *T) WaitForOrderAuthorized(ctx context.Context) (string, error)
```
WaitForOrderAuthorized waits until an order is authorized and returns its
order ID.


```go
func (p *T) WaitForReady(ctx context.Context) error
```







