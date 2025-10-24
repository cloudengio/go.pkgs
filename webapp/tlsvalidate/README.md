# Package [cloudeng.io/webapp/tlsvalidate](https://pkg.go.dev/cloudeng.io/webapp/tlsvalidate?tab=doc)

```go
import cloudeng.io/webapp/tlsvalidate
```

Package tlsvalidate provides functions for validating TLS certificates
across multiple hosts and addresses.

## Types
### Type Option
```go
type Option func(o *options)
```
Option represents an option for configuring a Validator.

### Functions

```go
func WithCheckSerialNumbers(check bool) Option
```
WithCheckSerialNumbers returns an option that configures the validator to
check that the certificates for all IP addresses for a given host have the
same serial number.


```go
func WithCiphersuites(suites []uint16) Option
```
WithCiphersuites returns an option that configures the validator to check
that the ciphersuite used is one of the specified ciphersuites.


```go
func WithExpandDNSNames(expand bool) Option
```
WithExpandDNSNames returns an option that configures the validator to expand
the supplied hostname to all of its IP addresses. If false, the hostname is
used as is.


```go
func WithIPv4Only(ipv4Only bool) Option
```
WithIPv4Only returns an option that configures the validator to only
consider IPv4 addresses for a host.


```go
func WithIssuerRegexps(exprs ...*regexp.Regexp) Option
```
WithIssuerRegexps returns an option that configures the validator to check
that the certificate's issuer matches at least one of the provided regular
expressions.


```go
func WithRootCAs(rootCAs *x509.CertPool) Option
```
WithRootCAs returns an option that configures the validator to use the
supplied pool of root CAs for verification.


```go
func WithTLSMinVersion(version uint16) Option
```
WithTLSMinVersion returns an option that configures the validator to check
that the TLS version used is at least the specified version.


```go
func WithValidForAtLeast(validFor time.Duration) Option
```
WithValidForAtLeast returns an option that configures the validator to check
that the certificate is valid for at least the specified duration.




### Type Validator
```go
type Validator struct {
	// contains filtered or unexported fields
}
```
Validator provides a way to validate TLS certificates.

### Functions

```go
func NewValidator(opts ...Option) *Validator
```
NewValidator returns a new Validator configured with the supplied options.



### Methods

```go
func (v *Validator) Validate(ctx context.Context, host, port string) error
```
Validate performs TLS validation for the given host and port. It may expand
the host to multiple IP addresses and will validate each one concurrently.







