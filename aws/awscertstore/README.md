# Package [cloudeng.io/aws/awscertstore](https://pkg.go.dev/cloudeng.io/aws/awscertstore?tab=doc)

```go
import cloudeng.io/aws/awscertstore
```

Package awscertstore provides an implementation of a autocert.DirCache and
cloudeng.io/webapp.CertStore for use when managing TLS certificates on AWS.
In particular, it uses the AWS secrets manager to store TLS certificates.

## Variables
### ErrUnsupportedOperation, ErrCacheMiss
```go
// ErrUnsupportedOperation is returned for any unsupported operations.
ErrUnsupportedOperation = errors.New("unsupported operation")
// ErrCacheMiss is the same as autocert.ErrCacheMiss
ErrCacheMiss = autocert.ErrCacheMiss

```

### AutoCertStore
```go
// AutoCertStore creates instances of webapp.CertStore using
// NewHybridCache.
AutoCertStore = CertStoreFactory{awsCacheName}

```



## Functions
### Func NewAWSCache
```go
func NewAWSCache(opts ...AWSCacheOption) autocert.Cache
```
NewAWSCache returns an instance of autocert.Cache that uses the AWS
secretsmanager. It assumes that a secret has already been created for
storing a given certificate and that the name of the certificate is the same
as the name of the secret.

### Func NewHybridCache
```go
func NewHybridCache(dir string, opts ...AWSCacheOption) autocert.Cache
```
NewHybridCache returns an instance of autocert.Cache that will store
certificates in 'backing' store, but use the local file system for
temporary/private data such as the ACME client's private key. This allows
for certificates to be shared across multiple hosts by using a distributed
'backing' store such as AWS' secretsmanager.



## Types
### Type AWSCacheOption
```go
type AWSCacheOption func(a *awscache)
```
AWSCacheOption represents an option to NewAWSCache.

### Functions

```go
func WithAWSConfig(cfg aws.Config) AWSCacheOption
```
WithAWSConfig specifies the aws.Config to use, it must be used to specify
the aws.Config to use for operations on the underlying secrets manager.




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
func (f CertStoreFactory) New(_ context.Context, _ string, opts ...interface{}) (webapp.CertStore, error)
```
New implements webapp.CertStoreFactory.


```go
func (f CertStoreFactory) Type() string
```
Type implements webapp.CertStoreFactory.







