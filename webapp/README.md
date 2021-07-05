# Package [cloudeng.io/webapp](https://pkg.go.dev/cloudeng.io/webapp?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/webapp)](https://goreportcard.com/report/cloudeng.io/webapp)

```go
import cloudeng.io/webapp
```

Package webapp and its sub-packages provide support for building webapps.
This includes utility routines for managing http.Server instances,
generating self-signed TLS certificates etc. The sub-packages provide
support for managing the assets to be served, various forms of
authentication and common toolchains such as webpack. For production
purposes assets are built into the server's binary, but for development they
are built into the binary but can be overridden from a local filesystem or
from a running development server that manages those assets (eg. a webpack
dev server instance). This provides the flexibility for both simple
deployment of production servers and iterative development within the same
application.

An example/template can be found in cmd/webapp.

## Functions
### Func NewHTTPServer
```go
func NewHTTPServer(addr string, handler http.Handler) (net.Listener, *http.Server, error)
```
NewHTTPServer returns a new *http.Server and a listener whose address
defaults to ":http".

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

### Func NewTLSServer
```go
func NewTLSServer(addr string, handler http.Handler) (net.Listener, *http.Server, error)
```
NewTLSServer returns a new *http.Server and a listener whose address
defaults to ":https".

### Func RedirectPort80
```go
func RedirectPort80(ctx context.Context, httpsAddr string) error
```
RedirectPort80 starts an http.Server that will redirect port 80 to the
specified supplied https port.

### Func RedirectToHTTPSHandlerFunc
```go
func RedirectToHTTPSHandlerFunc(tlsPort string) http.HandlerFunc
```
RedirectToHTTPSHandlerFunc is a http.HandlerFunc that will redirect to the
specified port but using https as the scheme. Install it on port 80 to
redirect all http requests to https on tlsPort. tlsPort defaults to 443.

### Func SelfSignedCertCommand
```go
func SelfSignedCertCommand(name string) *subcmd.Command
```

### Func ServeTLSWithShutdown
```go
func ServeTLSWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, cert, key string, grace time.Duration) error
```
ServeTLSWithShutdown is like ServeWithShutdown except for a TLS server.

### Func ServeWithShutdown
```go
func ServeWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error
```
ServeWithShutdown runs srv.ListenAndServe in background and then waits for
the context to be canceled. It will then attempt to shutdown the web server
within the specified grace period.



## Types
### Type HTTPServerFlags
```go
type HTTPServerFlags struct {
	Address         string `subcmd:"https,:8080,address to run https web server on"`
	CertificateFile string `subcmd:"ssl-cert,localhost.crt,ssl certificate file"`
	KeyFile         string `subcmd:"ssl-key,localhost.key,ssl private key file"`
}
```
HTTPServerFlags defines commonly used flags for running an http server.


### Type SelfSignedOption
```go
type SelfSignedOption func(ssc *selfSignedCertOptions)
```

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







