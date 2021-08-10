# Package [cloudeng.io/webapp/webauth/acme](https://pkg.go.dev/cloudeng.io/webapp/webauth/acme?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/webapp/webauth/acme)](https://goreportcard.com/report/cloudeng.io/webapp/webauth/acme)

```go
import cloudeng.io/webapp/webauth/acme
```

Package acme provides support for working with acme/letsencrypt providers.

## Constants
### LetsEncryptStaging, LetsEncryptProduction
```go
// LetsEncryptStaging is the URL for letsencrypt.org's staging service
// and is used as the default by this package.
LetsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
// LetsEncryptProduction is the URL for letsencrypt.org's production service.
LetsEncryptProduction = acme.LetsEncryptURL

```



## Variables
### AutoCertDiskStore, AutoCertNullStore
```go
// AutoCertDiskStore creates instances of webapp.CertStore using
// NewDirCache with read-only set to true.
AutoCertDiskStore = CertStoreFactory{dirCacheName}
// AutoCertNullStore creates instances of webapp.CertStore using
// NewNullCache.
AutoCertNullStore = CertStoreFactory{nullCacheName}

```

### ErrCacheMiss
```go
ErrCacheMiss = autocert.ErrCacheMiss

```
ErrCacheMiss is the same as autocert.ErrCacheMiss



## Functions
### Func NewDirCache
```go
func NewDirCache(dir string, readonly bool) autocert.Cache
```
NewDirCache returns an instance of a local filesystem based cache for
certificates and the acme account key but with file system locking. Set the
readonly argument for readonly access via the 'Get' method, this will
typically be used to safely extract keys for use by other servers. However,
ideally, a secure shared services such as Amazon's secrets manager should be
used instead.

### Func NewManagerFromFlags
```go
func NewManagerFromFlags(ctx context.Context, cache autocert.Cache, cl CertFlags) (*autocert.Manager, error)
```
NewManagerFromFlags creates a new autocert.Manager from the flag values. The
cache may be not be nil.

### Func NewNullCache
```go
func NewNullCache() autocert.Cache
```
NewNullCache returns an autocert.Cache that never stores any data and is
intended for use when testing.



## Types
### Type CertFlags
```go
type CertFlags struct {
	AcmeClientHost string          `subcmd:"acme-client-host,,'host running the acme client responsible for refreshing certificates, https requests to this host for one of the certificate hosts will result in the certificate for the certificate host being refreshed if necessary'"`
	Hosts          flags.Repeating `subcmd:"acme-cert-host,,'host for which certs are to be obtained'"`
	AcmeProvider   string          `subcmd:"acme-service,letsencrypt-staging,'the acme service to use, specify letsencrypt or letsencrypt-staging or a url'"`
	RenewBefore    time.Duration   `subcmd:"acme-renew-before,720h,how early certificates should be renewed before they expire."`
	Email          string          `subcmd:"acme-email,,email to contact for information on the domain"`
	TestingCAPem   string          `subcmd:"acme-testing-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
}
```
CertFlags represents the flags required to configure an autocert.Manager
isntance for managing TLS certificates for hosts/domains using the acme
http-01 challenge. Note that wildcard domains are not supported by this
challenge. The currently supported/tested acme service providers are
letsencrypt staging and production via the values 'letsencrypt-staging' and
'letsencrypt' for the --acme-service flag; however any URL can be specified
via this flag.


### Type CertStoreFactory
```go
type CertStoreFactory struct {
	// contains filtered or unexported fields
}
```
CertStoreFactory represents the webapp.CertStore's that can be created by
this package.

### Methods

```go
func (f CertStoreFactory) Describe() string
```
Describe implements webapp.CertStoreFactory.


```go
func (f CertStoreFactory) New(ctx context.Context, dir string) (webapp.CertStore, error)
```
New implements webapp.CertStoreFactory.


```go
func (f CertStoreFactory) Type() string
```
Type implements webapp.CertStoreFactory.







