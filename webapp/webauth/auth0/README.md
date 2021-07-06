# Package [cloudeng.io/webapp/webauth/auth0](https://pkg.go.dev/cloudeng.io/webapp/webauth/auth0?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/webapp/webauth/auth0)](https://goreportcard.com/report/cloudeng.io/webapp/webauth/auth0)

```go
import cloudeng.io/webapp/webauth/auth0
```


## Types
### Type Authenticator
```go
type Authenticator struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewAuthenticator(domain, audience string, opts ...Option) (*Authenticator, error)
```



### Methods

```go
func (a *Authenticator) CheckJWT(token string) error
```




### Type JWKS
```go
type JWKS struct {
	*jose.JSONWebKeySet
}
```
JWKS represents the KWT Key Set returned by auth0.com. See
https://auth0.com/docs/tokens/json-web-tokens/json-web-key-set-properties

### Functions

```go
func JWKSForDomain(tenant string) (*JWKS, error)
```




### Type Option
```go
type Option func(*Authenticator)
```

### Functions

```go
func RS256() Option
```


```go
func StaticJWKS(jwks *JWKS) Option
```







