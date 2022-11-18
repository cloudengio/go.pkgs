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
are built into the binary but can be overridden from a local filesystem
or from a running development server that manages those assets (eg.
a webpack dev server instance). This provides the flexibility for both
simple deployment of production servers and iterative development within the
same application.

An example/template can be found in cmd/webapp.

## Functions
### Func CertPoolForTesting
```go
func CertPoolForTesting(pemFiles ...string) (*x509.CertPool, error)
```
CertPoolForTesting returns a new x509.CertPool containing the certs in the
specified pem files. It is intended for testing purposes only.

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
func NewTLSServer(addr string, handler http.Handler, cfg *tls.Config) (net.Listener, *http.Server, error)
```
NewTLSServer returns a new *http.Server and a listener whose address
defaults to ":https".

### Func RedirectPort80
```go
func RedirectPort80(ctx context.Context, httpsAddr string, acmeRedirectHost string) error
```
RedirectPort80 starts an http.Server that will redirect port 80 to the
specified supplied https port. If acmeRedirect is specified then acme
HTTP-01 challenges will be redirected to that URL.

### Func RedirectToHTTPSHandlerFunc
```go
func RedirectToHTTPSHandlerFunc(tlsPort string, acmeRedirectHost *url.URL) http.HandlerFunc
```
RedirectToHTTPSHandlerFunc is a http.HandlerFunc that will redirect to
the specified port but using https as the scheme. Install it on port 80 to
redirect all http requests to https on tlsPort. tlsPort defaults to 443.
If acmeRedirect is specified then acme HTTP-01 challenges will be redirected
to that URL.

### Func RegisterCertStoreFactory
```go
func RegisterCertStoreFactory(cache CertStoreFactory)
```
RegisterCertStoreFactory makes the supplied CertStoreFactory available for
use via the TLSCertStoreFlags command line flags.

### Func RegisteredCertStores
```go
func RegisteredCertStores() []string
```
RegisteredCertStores returns the list of currently registered certificate
stores.

### Func SelfSignedCertCommand
```go
func SelfSignedCertCommand(name string) *subcmd.Command
```
SelfSignedCertCommand returns a subcmd.Command that provides the ability to
generate a self signed certificate and private key file.

### Func ServeTLSWithShutdown
```go
func ServeTLSWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error
```
ServeTLSWithShutdown is like ServeWithShutdown except for a TLS server.
Note that any TLS options must be configured prior to calling this function
via the TLSConfig field in http.Server.

### Func ServeWithShutdown
```go
func ServeWithShutdown(ctx context.Context, ln net.Listener, srv *http.Server, grace time.Duration) error
```
ServeWithShutdown runs srv.ListenAndServe in background and then waits for
the context to be canceled. It will then attempt to shutdown the web server
within the specified grace period.

### Func TLSConfigFromFlags
```go
func TLSConfigFromFlags(ctx context.Context, cl HTTPServerFlags, storeOpts ...interface{}) (*tls.Config, error)
```
TLSConfigFromFlags creates a tls.Config based on the supplied flags,
which may require obtaining certificates directly from pem files or
from a possibly remote certificate store using TLSConfigUsingCertStore.
Any supplied storeOpts are passed to TLSConfigUsingCertStore.

### Func TLSConfigUsingCertFiles
```go
func TLSConfigUsingCertFiles(certFile, keyFile string) (*tls.Config, error)
```
TLSConfigUsingCertFiles returns a tls.Config configured with the certificate
read from the supplied files.

### Func TLSConfigUsingCertStore
```go
func TLSConfigUsingCertStore(ctx context.Context, typ, name, testingCA string, storeOpts ...interface{}) (*tls.Config, error)
```
TLSConfigUsingCertStore returns a tls.Config configured with the certificate
obtained from a certificate store.



## Types
### Type CertServingCache
```go
type CertServingCache struct {
	// contains filtered or unexported fields
}
```
CertServingCache implements an in-memory cache of TLS/SSL certificates
loaded from a backing store. Validation of the certificates is performed
on loading rather than every use. It provides a GetCertificate method that
can be used by tls.Config. A TTL (default of 6 hours) is used so that the
in-memory cache will reload certificates from the store on a periodic basis
(with some jitter) to allow for certificates to be refreshed.

### Functions

```go
func NewCertServingCache(certStore CertStore, opts ...CertServingCacheOption) *CertServingCache
```
NewCertServingCache returns a new instance of CertServingCache that uses the
supplied CertStore.



### Methods

```go
func (m *CertServingCache) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
```
GetCertificate can be assigned to tls.Config.GetCertificate.




### Type CertServingCacheOption
```go
type CertServingCacheOption func(*CertServingCache)
```
CertServingCacheOption represents options to NewCertServingCache.

### Functions

```go
func CertCacheNowFunc(fn func() time.Time) CertServingCacheOption
```
CertCacheNowFunc sets the function used to obtain the current time. This is
generally only required for testing purposes.


```go
func CertCacheRootCAs(rootCAs *x509.CertPool) CertServingCacheOption
```
CertCacheRootCAs sets the rootCAs to be used when verifying the validity of
the certificate loaded from the back store.


```go
func CertCacheTTL(ttl time.Duration) CertServingCacheOption
```
CertCacheTTL sets the in-memory TTL beyond which cache entries are
refreshed. This is generally only required for testing purposes.




### Type CertStore
```go
type CertStore interface {
	Get(ctx context.Context, name string) ([]byte, error)
	Put(ctx context.Context, name string, data []byte) error
	Delete(ctx context.Context, name string) error
}
```
CertStore represents a store for TLS certificates.

### Functions

```go
func NewCertStore(ctx context.Context, typ, name string, storeOpts ...interface{}) (CertStore, error)
```




### Type CertStoreFactory
```go
type CertStoreFactory interface {
	Type() string
	Describe() string
	New(ctx context.Context, name string, opts ...interface{}) (CertStore, error)
}
```
CertStoreFactory is the interface that must be implemented to register
a new CertStore type with this package so that it may accessed via the
TLSCertStoreFlags command line flags.


### Type HTTPServerFlags
```go
type HTTPServerFlags struct {
	Address string `subcmd:"https,:8080,address to run https web server on"`
	TLSCertFlags
	AcmeRedirectTarget string `subcmd:"acme-redirect-target,,host implementing acme client that this http server will redirect acme challenges to"`
	TestingCAPem       string `subcmd:"acme-testing-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
}
```
HTTPServerFlags defines commonly used flags for running an http server.
TLS certificates may be retrieved either from a local cert and key file as
specified by tls-cert and tls-key; this is generally used for testing or
when the domain certificates are available only as files. The altnerative,
preferred for production, source for TLS certificates is from a cache as
specified by tls-cert-cache-type and tls-cert-cache-name. The cache may be
on local disk, or preferably in some shared service such as Amazon's Secrets
Service.


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




### Type TLSCertFlags
```go
type TLSCertFlags struct {
	TLSCertStoreFlags
	CertificateFile string `subcmd:"tls-cert,,ssl certificate file"`
	KeyFile         string `subcmd:"tls-key,,ssl private key file"`
}
```
TLSCertFlags defines commonly used flags for obtaining TLS/SSL certificates.
Certificates may be obtained in one of two ways: from a cache of
certificates, or from local files.


### Type TLSCertStoreFlags
```go
type TLSCertStoreFlags struct {
	CertStoreType  string `subcmd:"tls-cert-store-type,,'the type of the certificate store to use for retrieving tls certificates, use --tls-list-stores to see the currently available types'"`
	CertStore      string `subcmd:"tls-cert-store,,'name/address of the certificate cache to use for retrieving tls certificates, the interpreation of this depends on the tls-cert-store-type flag'"`
	ListStoreTypes bool   `subcmd:"tls-list-stores,,list the available types of tls certificate store"`
}
```
TLSCertStoreFlags defines commonly used flags for specifying a
TLS/SSL certificate store. This is generally used in conjunction with
TLSConfigFromFlags for apps that simply want to use stored certificates.
Apps that manage/obtain/renew certificates may use them directly.





