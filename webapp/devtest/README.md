# Package [cloudeng.io/webapp/devtest](https://pkg.go.dev/cloudeng.io/webapp/devtest?tab=doc)

```go
import cloudeng.io/webapp/devtest
```

Package devtest provides utilities for the development and testing of web
applications, including TLS certificate generation and management.

## Functions
### Func CertPoolForTesting
```go
func CertPoolForTesting(pemFiles ...string) (*x509.CertPool, error)
```
CertPoolForTesting returns a new x509.CertPool containing the certs in the
specified pem files. If no pem files are specified nil it will return the
system cert pool. It is intended for testing purposes only.

### Func NewSelfSignedCert
```go
func NewSelfSignedCert(certFile, keyFile string, options ...SelfSignedOption) error
```
NewSelfSignedCert creates a self signed certificate. Default values for the
supported options are:
  - an rsa 4096 bit private key will be generated and used.
  - "localhost" and "127.0.0.1" are used for the DNS and IP addresses
    included in the certificate.
  - certificates are valid from time.Now() and for 5 days.
  - the organization is 'cloudeng llc'.

### Func NewSelfSignedCertUsingMkcert
```go
func NewSelfSignedCertUsingMkcert(certFile, keyFile string, hosts ...string) error
```
NewSelfSignedCertUsingMkcert uses mkcert
(https://github.com/FiloSottile/mkcert) to create certificates. If mkcert
--install has been run then these certificates will be trusted by the
browser and other local applications.



## Types
### Type JSServer
```go
type JSServer struct {
	// contains filtered or unexported fields
}
```
JSServer provides a http.Handler for serving JavaScript files using a simple
template that executes each file in turn. An optional TypescriptSources can
be provided to compile TypeScript files before generating the http response.

### Functions

```go
func NewJSServer(title string, ts *TypescriptSources, jsScripts ...string) *JSServer
```



### Methods

```go
func (jss *JSServer) ServeJS(rw http.ResponseWriter, r *http.Request)
```
ServeJS handles HTTP requests for serving a series of Javascript files.




### Type Pebble
```go
type Pebble struct {
	// contains filtered or unexported fields
}
```
Pebble manages a pebble instance for testing purposes.

### Functions

```go
func NewPebble(binary string) *Pebble
```
NewPebble creates a new Pebble instance. The supplied configFile will
be used to configure the pebble instance. The server is not started by
NewPebble.



### Methods

```go
func (p *Pebble) PID() int
```
PID returns the process ID of the pebble instance.


```go
func (p *Pebble) Start(ctx context.Context, dir, cfg string, forward io.WriteCloser) error
```
Start the pebble instance with its output forwarded to the supplied writer.


```go
func (p *Pebble) Stop()
```
Stop the pebble instance.


```go
func (p *Pebble) WaitForIssuedCertificateSerial(ctx context.Context) (string, error)
```
WaitForIssuedCertificateSerial waits until a certificate is issued and
returns its serial number.


```go
func (p *Pebble) WaitForReady(ctx context.Context) error
```




### Type PebbleConfig
```go
type PebbleConfig struct {
	Address           string
	ManagementAddress string
	HTTPPort          int
	TLSPort           int
	Certificate       []byte
	Key               []byte
	TestCertBase      string
	RootCertURL       string
	// contains filtered or unexported fields
}
```
PebbleConfig represents the configuration for a pebble instance that's
relevant to using it for testing clients.

### Functions

```go
func NewPebbleConfig() (PebbleConfig, error)
```
NewPebbleConfig creates a new PebbleConfig instance with default values.



### Methods

```go
func (pc PebbleConfig) CA() *x509.CertPool
```
CA returns a CertPool containing the root pebble CA certificate. Use it when
configuring clients to connect to the pebble instance.


```go
func (pc *PebbleConfig) CreateCertsAndUpdateConfig(ctx context.Context, outputDir string) (string, error)
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
func (pc PebbleConfig) GetIssuingCA(ctx context.Context) (*x509.CertPool, error)
```
IssuingCA returns a CertPool containing the issuing CA certificate used by
pebble to sign issued certificates.


```go
func (pc PebbleConfig) GetIssuingCert(ctx context.Context) ([]byte, error)
```
GetIssuingCert retrieves the pebble certificate, including intermediates,
used to sign issued certificates.




### Type SelfSignedOption
```go
type SelfSignedOption func(ssc *selfSignedCertOptions)
```
SelfSignedOption represents an option to NewSelfSignedCert.

### Functions

```go
func CertAllIPAddresses() SelfSignedOption
```
CertIPAddresses specifies that all local IPs be used in the generated
certificate.


```go
func CertDNSHosts(hosts ...string) SelfSignedOption
```
CertDNSHosts specifies the set of dns host names to use in the generated
certificate.


```go
func CertIPAddresses(ips ...string) SelfSignedOption
```
CertIPAddresses specifies the set of ip addresses to use in the generated
certificate.


```go
func CertOrganizations(orgs ...string) SelfSignedOption
```
CertOrganizations specifies that the organization to be used in the
generated certificate.


```go
func CertPrivateKey(key crypto.PrivateKey) SelfSignedOption
```
CertPrivateKey specifies the private key to use for the certificate.




### Type TypescriptOption
```go
type TypescriptOption func(o *typescriptOptions)
```
TypescriptOption represents an option to NewTypescriptSources.

### Functions

```go
func WithTypescriptCompiler(compiler string) TypescriptOption
```
WithTypescriptCompiler sets the TypeScript compiler to use. The default is
"tsc".


```go
func WithTypescriptTarget(target string) TypescriptOption
```
WithTypescriptTarget sets the target version for the TypeScript compiler.
The default is "es2015".




### Type TypescriptSources
```go
type TypescriptSources struct {
	// contains filtered or unexported fields
}
```
TypescriptSources represents a collection of TypeScript source files that
can be compiled using the TypeScript compiler.

### Functions

```go
func NewTypescriptSources(opts ...TypescriptOption) *TypescriptSources
```
NewTypescriptSources creates a new instance of TypescriptSources



### Methods

```go
func (ts *TypescriptSources) Compile(ctx context.Context) error
```
Compile compiles the TypeScript sources that have been modified since it was
last run.


```go
func (ts *TypescriptSources) SetDirAndFiles(dir string, files ...string)
```
SetFiles sets the directory and files for the TypeScript sources. The output
will be in the same directory, 'dir', as the input files.







