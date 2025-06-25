# Package [cloudeng.io/algo/digests](https://pkg.go.dev/cloudeng.io/algo/digests?tab=doc)

```go
import cloudeng.io/algo/digests
```

Package digests provides a simple interface to create and validate digests
using various algorithms such as SHA1, MD5, SHA256, and SHA512. Support is
provided for working with digests in both base64 and hex formats.

## Functions
### Func FromBase64
```go
func FromBase64(digest string) ([]byte, error)
```

### Func FromHex
```go
func FromHex(digest string) ([]byte, error)
```

### Func IsSupported
```go
func IsSupported(algo string) bool
```

### Func ParseHex
```go
func ParseHex(digest string) (algo, hexdigits string, err error)
```
ParseHex decodes a digest specification of the form <algo>=<hex-digits>.

### Func Supported
```go
func Supported() []string
```

### Func ToBase64
```go
func ToBase64(digest []byte) string
```

### Func ToHex
```go
func ToHex(digest []byte) string
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
func New(algo string, digest []byte) (Hash, error)
```
New creates a new Hash instance based on the specified algorithm. Currently
supported algorithms are "sha1", "md5", "sha256", and "sha512".

Note: MD5 and SHA1 are cryptographically weak and should not be used for
security-sensitive applications.



### Methods

```go
func (h Hash) Validate() bool
```
Validate checks if the hash instance's computed sum matches the expected
digest.







