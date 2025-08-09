# Package [cloudeng.io/webapp/webauth/jwtutil](https://pkg.go.dev/cloudeng.io/webapp/webauth/jwtutil?tab=doc)

```go
import cloudeng.io/webapp/webauth/jwtutil
```

Package jwtutil provides support for creating and verifying JSON Web Tokens
(JWTs) managed by the github.com/golang-jwt/jwt/v5 package. This package
provides simplified wrappers around the JWT signing and verification process
to allow for more convenient usage in web applications.

## Types
### Type ED25519PublicKey
```go
type ED25519PublicKey struct {
	// contains filtered or unexported fields
}
```
NewED25519PublicKey creates a new ED25519PublicKey instance with the given
public key and key ID.

### Functions

```go
func NewED25519PublicKey(pub ed25519.PublicKey, id string) *ED25519PublicKey
```



### Methods

```go
func (v ED25519PublicKey) KeyFunc(token *jwt.Token) (any, error)
```




### Type ED25519Signer
```go
type ED25519Signer struct {
	ED25519PublicKey
	// contains filtered or unexported fields
}
```
ED25519Signer implements the Signer interface using an Ed25519 private key.

### Functions

```go
func NewED25519Signer(pub ed25519.PublicKey, priv ed25519.PrivateKey, id string) ED25519Signer
```



### Methods

```go
func (s ED25519Signer) Sign(claims jwt.Claims) (string, error)
```




### Type PublicKeys
```go
type PublicKeys interface {
	KeyFunc(token *jwt.Token) (any, error)
}
```
PublicKeys represents an interface to the jwt.KeyFunc called by the JWT
parser to retrieve the public key or keys used for verifying JWTs.


### Type Signer
```go
type Signer interface {
	Sign(jwt.Claims) (string, error)
	PublicKeys
}
```





