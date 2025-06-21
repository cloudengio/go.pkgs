# Package [cloudeng.io/algo/hash](https://pkg.go.dev/cloudeng.io/algo/hash?tab=doc)

```go
import cloudeng.io/algo/hash
```

Package hash provides a simple interface to create and validate hashes
using various algorithms such as SHA1, MD5, SHA256, and SHA512. The hashes
are created from base64 encoded digests, which allows for easy storage and
transmission of hash values.

## Functions
### Func FromBase64
```go
func FromBase64(digest string) ([]byte, error)
```

### Func ToBase64
```go
func ToBase64(digest []byte) string
```



## Types
### Type Hash
```go
type Hash struct {
	hash.Hash
	Algo   string
	Digest []byte
}
```

### Functions

```go
func New(algo, digest string) (Hash, error)
```
New creates a new Hash instance based on the specified algorithm and digest.
Supported algorithms are "sha1", "md5", "sha256", and "sha512" and the
digest is base64 encoded.

Note: MD5 and SHA1 are cryptographically weak and should not be used for
security-sensitive applications.



### Methods

```go
func (h Hash) Validate() bool
```






## Examples
### [ExampleHash](https://pkg.go.dev/cloudeng.io/algo/hash?tab=doc#example-Hash)




