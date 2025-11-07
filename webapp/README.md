# Package [cloudeng.io/webapp](https://pkg.go.dev/cloudeng.io/webapp?tab=doc)

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

## Constants
### PreferredTLSMinVersion
```go
PreferredTLSMinVersion = tls.VersionTLS13

```
PreferredTLSMinVersion is the preferred minimum TLS version for tls.Config
instances created by this package.



## Variables
### PreferredCipherSuites
```go
PreferredCipherSuites = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
}

```
PreferredCipherSuites is the list of preferred cipher suites for tls.Config
instances created by this package.

### PreferredCurves
```go
PreferredCurves = []tls.CurveID{
	tls.X25519,
	tls.CurveP256,
}

```
PreferredCurves is the list of preferred elliptic curves for tls.Config
instances created by this package.

### PreferredSignatureSchemes
```go
PreferredSignatureSchemes = []tls.SignatureScheme{
	tls.ECDSAWithP256AndSHA256,
	tls.ECDSAWithP384AndSHA384,
	tls.ECDSAWithP521AndSHA512,
}

```
PreferredSignatureSchemes is the list of preferred signature schemes
generally used for obtainint TLS certificates.



## Functions
### Func FindLeafPEM
```go
func FindLeafPEM(certsPEM []*pem.Block) ([]byte, *x509.Certificate, error)
```
FindLeafPEM searches the supplied PEM blocks for the leaf certificate and
returns its DER encoding along with the parsed x509.Certificate.

### Func NewHTTPClient
```go
func NewHTTPClient(ctx context.Context, opts ...HTTPClientOption) (*http.Client, error)
```
NewHTTPClient creates a new HTTP client configured according to the
specified options.

### Func NewHTTPServer
```go
func NewHTTPServer(ctx context.Context, addr string, handler http.Handler) (net.Listener, *http.Server, error)
```
NewHTTPServer returns a new *http.Server using ParseAddrPortDefaults(addr,
"http") to obtain the address to listen on and NewHTTPServerOnly to create
the server.

### Func NewHTTPServerOnly
```go
func NewHTTPServerOnly(ctx context.Context, addr string, handler http.Handler) *http.Server
```
NewHTTPServerOnly returns a new *http.Server whose address defaults to
":http" and with it's BaseContext set to the supplied context. ErrorLog is
set to log errors via the ctxlog package.

### Func NewTLSServer
```go
func NewTLSServer(ctx context.Context, addr string, handler http.Handler, cfg *tls.Config) (net.Listener, *http.Server, error)
```
NewTLSServer returns a new *http.Server using ParseAddrPortDefaults(addr,
"https") to obtain the address to listen on and NewTLSServerOnly to create
the server.

### Func NewTLSServerOnly
```go
func NewTLSServerOnly(ctx context.Context, addr string, handler http.Handler, cfg *tls.Config) *http.Server
```
NewTLSServerOnly returns a new *http.Server whose address defaults to
":https" and with it's BaseContext set to the supplied context and TLSConfig
set to the supplied config. ErrorLog is set to log errors via the ctxlog
package.

### Func ParseAddrPortDefaults
```go
func ParseAddrPortDefaults(addr, port string) string
```
ParseAddrPortDefaults parses addr and returns an address:port string.
If addr does not contain a port then the supplied port is used.

### Func ParseCertsPEM
```go
func ParseCertsPEM(pemData []byte) ([]*x509.Certificate, error)
```
ParseCertsPEM parses certificates from the provided PEM data.

### Func ParsePEM
```go
func ParsePEM(pemData []byte) (privateKeys, publicKeys, certs []*pem.Block)
```
ParsePEM parses private keys and certificates from the provided PEM data.

### Func ParsePrivateKeyDER
```go
func ParsePrivateKeyDER(der []byte) (crypto.Signer, error)
```
ParsePrivateKeyDER parses a DER encoded private key. It tries PKCS#1,
PKCS#8 and then SEC 1 for EC keys.

### Func ReadAndParseCertsPEM
```go
func ReadAndParseCertsPEM(ctx context.Context, fs file.ReadFileFS, pemFile string) ([]*x509.Certificate, error)
```
ReadAndParseCertsPEM loads certificates from the specified PEM file.

### Func ReadAndParsePrivateKeyPEM
```go
func ReadAndParsePrivateKeyPEM(ctx context.Context, fs file.ReadFileFS, pemFile string) (crypto.Signer, error)
```
ReadAndParsePrivateKeyPEM reads and parses a PEM encoded private key from
the specified file.

### Func RedirectHandler
```go
func RedirectHandler(redirects ...Redirect) http.Handler
```

### Func RedirectPort80
```go
func RedirectPort80(ctx context.Context, redirects ...Redirect) error
```
RedirectPort80 starts an http.Server that will redirect port 80 to the
specified supplied https port. If acmeRedirect is specified then acme
HTTP-01 challenges will be redirected to that URL. The server will run in
the background until the supplied context is canceled.

### Func SafePath
```go
func SafePath(path string) error
```
SafePath checks if the given path is safe for use as a filename screening
for control characters, windows device names, relative paths, paths (eg.
a/b is not allowed) etc.

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

### Func SplitHostPort
```go
func SplitHostPort(hostport string) (string, string)
```
SplitHostPort splits hostport into host and port. If hostport does not
contain a port, then the returned port is empty. It assumes that the
hostport is a valid ipv4 or ipv6 address.

### Func TLSConfigUsingCertFiles
```go
func TLSConfigUsingCertFiles(certFile, keyFile string) (*tls.Config, error)
```
TLSConfigUsingCertFiles returns a tls.Config configured with the certificate
read from the supplied files.

### Func TLSConfigUsingCertStore
```go
func TLSConfigUsingCertStore(ctx context.Context, store autocert.Cache, cacheOpts ...CertServingCacheOption) (*tls.Config, error)
```
TLSConfigUsingCertStore returns a tls.Config configured with the
certificate obtained from the specified certificate store accessed via a
CertServingCache created with the supplied options.

### Func VerifyCertChain
```go
func VerifyCertChain(dnsname string, certs []*x509.Certificate, roots *x509.CertPool) ([][]*x509.Certificate, error)
```
VerifyCertChain verifies the supplied certificate chain using the provided
root certificates and verifies that the leaf certificate is valid for the
specified dnsname. It returns the verified chains on success.

### Func WaitForServers
```go
func WaitForServers(ctx context.Context, interval time.Duration, addrs ...string) error
```
WaitForServers waits for all supplied addresses to be available by
attempting to open a TCP connection to each address at the specified
interval.

### Func WaitForURLs
```go
func WaitForURLs(ctx context.Context, interval time.Duration, urls ...string) error
```
WaitForURLs waits for all supplied URLs to be available by attempting to
perform HTTP GET requests to each URL at the specified interval.



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
func NewCertServingCache(ctx context.Context, certStore autocert.Cache, opts ...CertServingCacheOption) *CertServingCache
```
NewCertServingCache returns a new instance of CertServingCache that uses
the supplied CertStore. The supplied context is cached and used by the
GetCertificate method, this allows for credentials etc to be passed to the
CertStore.Get method called by GetCertificate via the context.



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




### Type HTTPClientOption
```go
type HTTPClientOption func(o *httpClientOptions)
```
HTTPClientOption is used to configure an HTTP client.

### Functions

```go
func WithCustomCAPEMFile(caPEMFile string) HTTPClientOption
```
WithCustomCAPEMFile configures the HTTP client to use the specified custom
CA PEM data as a root CA.


```go
func WithCustomCAPool(caPool *x509.CertPool) HTTPClientOption
```
WithCustomCAPool configures the HTTP client to use the specified custom CA
pool. It takes precedence over WithCustomCAPEMFile.


```go
func WithTracingTransport(to ...httptracing.TraceRoundtripOption) HTTPClientOption
```
WithTracingTransport configures the HTTP client to use a tracing round
tripper with the specified options.




### Type HTTPServerConfig
```go
type HTTPServerConfig struct {
	Address  string        `yaml:"address,omitempty"`
	TLSCerts TLSCertConfig `yaml:"tls_certs,omitempty"`
}
```
HTTPServerConfig defines configuration for an http server.

### Methods

```go
func (hc HTTPServerConfig) TLSConfig() (*tls.Config, error)
```




### Type HTTPServerFlags
```go
type HTTPServerFlags struct {
	Address string `subcmd:"https,:8080,address to run https web server on"`
	TLSCertFlags
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

### Methods

```go
func (cl HTTPServerFlags) HTTPServerConfig() HTTPServerConfig
```
HTTPServerConfig returns an HTTPServerConfig based on the supplied flags.




### Type Redirect
```go
type Redirect struct {
	Prefix string
	Target RedirectTarget
}
```
Redirect defines a URL path prefix which will be redirected to the specified
target.

### Functions

```go
func RedirectAcmeHTTP01(host string) Redirect
```
RedirectAcmeHTTP01 returns a Redirect that will redirect ACME HTTP-01
challenges to the specified host.


```go
func RedirectToHTTPSPort(addr string) Redirect
```
RedirectToHTTPSPort returns a Redirect that will redirect to the specified
address using https but with the following defaults: - if addr does not
contain a host then the host from the request is used - if addr does not
contain a port then port 443 is used.




### Type RedirectTarget
```go
type RedirectTarget func(*http.Request) (string, int)
```
RedirectTarget is a function that given an http.Request returns the target
URL for the redirect and the HTTP status code to use.


### Type TLSCertConfig
```go
type TLSCertConfig struct {
	CertFile string `yaml:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
}
```
TLSCertConfig defines configuration for TLS certificates obtained from local
files.

### Methods

```go
func (tc TLSCertConfig) TLSConfig() (*tls.Config, error)
```
TLSConfig returns a tls.Config.




### Type TLSCertFlags
```go
type TLSCertFlags struct {
	CertFile string `subcmd:"tls-cert,,tls certificate file"`
	KeyFile  string `subcmd:"tls-key,,tls private key file"`
}
```
TLSCertFlags defines commonly used flags for obtaining TLS/SSL certificates.
Certificates may be obtained in one of two ways: from a cache of
certificates, or from local files.

### Methods

```go
func (cl TLSCertFlags) TLSCertConfig() TLSCertConfig
```
Config returns a TLSCertConfig based on the supplied flags.






## Examples
### [ExampleServeWithShutdown](https://pkg.go.dev/cloudeng.io/webapp?tab=doc#example-ServeWithShutdown)




