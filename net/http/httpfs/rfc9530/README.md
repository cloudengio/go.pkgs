# Package [cloudeng.io/net/http/httpfs/rfc9530](https://pkg.go.dev/cloudeng.io/net/http/httpfs/rfc9530?tab=doc)

```go
import cloudeng.io/net/http/httpfs/rfc9530
```

Package rfc9530 provides utilities for working with RFC 9530. It includes
functions for parsing the Repr-Digest header as defined in RFC 9530. The
Repr-Digest header is used to convey the digest values of representations in
a format that allows multiple algorithms to be specified.

## Constants
### ReprDigestHeader
```go
ReprDigestHeader = "Repr-Digest"

```



## Functions
### Func AsHeaderValue
```go
func AsHeaderValue(algo, base64Digest string) string
```

### Func ChooseDigest
```go
func ChooseDigest(digests map[string]string, algos ...string) (string, bool)
```
ChooseDigest selects a digest from the provided map of digests based on
the specified algorithms with an indication of whether the returned digest
matches one of the requested algorithms. It checks the provided algorithms
in order and returns the first matching algorithm's digest if found.
If no algorithms match, it returns the first available digest in the map
based on the alphabetical order of the keys and a boolean indicating that
no requested algorithm was matched. The returned value is in the format
"algo=base64Digest" suitable for the Repr-Digest header.

### Func ParseAlgoDigest
```go
func ParseAlgoDigest(value string) (algo, base64Digest string, bytes []byte, err error)
```

### Func ParseReprDigest
```go
func ParseReprDigest(headerValue string) (map[string]string, error)
```
ParseReprDigest parses the Repr-Digest header value. It returns a map
of algorithm-to-digest mappings or an error if the format is invalid.
The digest value is the raw base64 string from the header.




