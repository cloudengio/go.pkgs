# Package [cloudeng.io/webapp/webauth/acme](https://pkg.go.dev/cloudeng.io/webapp/webauth/acme?tab=doc)

```go
import cloudeng.io/webapp/webauth/acme
```

Package acme provides support for working with acme/letsencrypt providers.

## Constants
### LetsEncryptStaging, LetsEncryptProduction
```go
// LetsEncryptStaging is the URL for the letsencrypt.org staging service
// and is used as the default by this package.
LetsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
// LetsEncryptProduction is the URL for the letsencrypt.org production service.
LetsEncryptProduction = acme.LetsEncryptURL

```



## Variables
### ErrReadonlyCache, ErrLocalOperation, ErrBackingOperation, ErrLockFailed
```go
ErrReadonlyCache = errors.New("readonly cache")
ErrLocalOperation = errors.New("local operation")
ErrBackingOperation = errors.New("backing store operation")
ErrLockFailed = errors.New("lock acquisition failed")

```

### ErrCacheMiss
```go
ErrCacheMiss = autocert.ErrCacheMiss

```
ErrCacheMiss is the same as autocert.ErrCacheMiss



## Functions
### Func IsAcmeAccountKey
```go
func IsAcmeAccountKey(name string) bool
```
IsAcmeAccountKey returns true if the specified name is for an ACME account
private key.

### Func IsLocalName
```go
func IsLocalName(name string) bool
```
IsLocalName returns true if the specified name is for local-only data such
as ACME client private keys or http-01 challenge tokens.

### Func NewManager
```go
func NewManager(ctx context.Context, cache autocert.Cache, cl ManagerConfig, allowedHosts ...string) (*autocert.Manager, error)
```
NewManager creates a new autocert.Manager from the supplied config. Any
supplied hosts, along with the ClientHost, are used to specify the allowed
hosts for the manager.

### Func RefreshCertificate
```go
func RefreshCertificate(ctx context.Context, mgr *autocert.Manager, host string) (*tls.Certificate, error)
```
RefreshCertificate attempts to refresh the certificate for the specified
host using the provided autocert.Manager by simulating a TLS ClientHello
for the specified host. It prefers to use the PreferredCipherSuites and
PreferredSignatureSchemes defined in webapp package to force the use of
ECDSA certificates rather than RSA.

### Func WrapHostPolicyNoPort
```go
func WrapHostPolicyNoPort(existing autocert.HostPolicy) autocert.HostPolicy
```
WrapHostPolicyNoPort wraps an existing autocert.HostPolicy to strip any port
information from the host before passing it to the existing policy. This is
required when running in a test environment where well-known/hardwired ports
(80, 443) are not used.



## Types
### Type AcmeFlags
```go
type AcmeFlags struct {
	ClientHost  string        `subcmd:"acme-client-host,,'host running the acme client responsible for refreshing certificates, https requests to this host for one of the certificate hosts will result in the certificate for the certificate host being refreshed if necessary'"`
	Provider    string        `subcmd:"acme-service,letsencrypt-staging,'the acme service to use, specify letsencrypt or letsencrypt-staging or a url'"`
	RenewBefore time.Duration `subcmd:"acme-renew-before,720h,how early certificates should be renewed before they expire."`
	Email       string        `subcmd:"acme-email,,email to contact for information on the domain"`
	UserAgent   string        `subcmd:"acme-user-agent,cloudeng.io/webapp/webauth/acme,'user agent to use when connecting to the acme service'"`
}
```
AcmeFlags represents the flags required to configure an autocert.Manager
isntance for managing TLS certificates for hosts/domains using the acme
http-01 challenge. Note that wildcard domains are not supported by this
challenge. The currently supported/tested acme service providers are
letsencrypt staging and production via the values 'letsencrypt-staging' and
'letsencrypt' for the --acme-service flag; however any URL can be specified
via this flag, in particular to use pebble for testing set this to the URL
of the local pebble instance and also set the --acme-testing-ca flag to
point to the pebble CA certificate pem file.

### Methods

```go
func (f AcmeFlags) ManagerConfig() ManagerConfig
```
Config converts the flag values to a Config instance.




### Type CacheFS
```go
type CacheFS interface {
	ReadFileCtx(ctx context.Context, name string) ([]byte, error)
	WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
	Delete(ctx context.Context, name string) error
}
```
CacheFS defines an interface that combines reading, writing and deleting
files and is used to create an acme/autocert cache.

### Functions

```go
func NewLocalStore(dir string) (CacheFS, error)
```




### Type CachingStore
```go
type CachingStore struct {
	// contains filtered or unexported fields
}
```
CachingStore implements a 'caching store' that intergrates withletsencrypt's
autocert. It provides an instance of autocert.Cache that will store
certificates in 'backing' store, but use the local file system for
temporary/private data such as the ACME client's private key. This allows
for certificates to be shared across multiple hosts by using a distributed
'backing' store such as AWS' secretsmanager. In addition, certificates may
be extracted safely on the host that manages them programmatically.

### Functions

```go
func NewCachingStore(localDir string, backingStore CacheFS, opts ...Option) (*CachingStore, error)
```
NewCachingStore returns an instance of autocert.Cache that will store
certificates in 'backing' store, but use the local file system for
temporary/private data such as the ACME client's private key. This allows
for certificates to be shared across multiple hosts by using a distributed
'backing' store such as AWS' secretsmanager. Certificates may be extracted
safely for use by other servers by using the readonly option. CachingStore
implements autocert.Cache.



### Methods

```go
func (dc *CachingStore) Delete(ctx context.Context, name string) error
```
Delete implements autocert.Cache.


```go
func (dc *CachingStore) Get(ctx context.Context, name string) ([]byte, error)
```
Get implements autocert.Cache.


```go
func (dc *CachingStore) Put(ctx context.Context, name string, data []byte) error
```
Put implements autocert.Cache.




### Type ManagerConfig
```go
type ManagerConfig struct {
	// Contact email for the ACME account, note, changing this may create
	// a new account with the ACME provider. The key associated with an account
	// is required for revoking certificates issued using that account.
	Email       string        `yaml:"email"`
	UserAgent   string        `yaml:"user_agent"`    // User agent to use when connecting to the ACME service.
	Provider    string        `yaml:"acme_provider"` // ACME service provider URL or 'letsencrypt' or 'letsencrypt-staging'.
	RenewBefore time.Duration `yaml:"renew_before"`  // How early certificates should be renewed before they expire.
	ClientHost  string        `yaml:"client_host"`   // Host running the ACME client responsible for refreshing certificates, always added to allowed hosts by NewManager.
}
```
ManagerConfig represents the configuration required to create an
autocert.Manager instance for managing TLS certificates for hosts/domains
using the acme http-01 challenge. Note that wildcard domains are not
supported by this challenge.


### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithReadonly(readonly bool) Option
```
WithReadonly sets whether the caching store is readonly.


```go
func WithSaveAccountKey(name string) Option
```
WithSaveAccountKey sets whether ACME account keys are to be saved to the
backing store using the specified name.







