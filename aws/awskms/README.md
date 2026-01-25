# Package [cloudeng.io/aws/awskms](https://pkg.go.dev/cloudeng.io/aws/awskms?tab=doc)

```go
import cloudeng.io/aws/awskms
```


## Functions
### Func NewSigner
```go
func NewSigner(ctx context.Context, client Client, keyID, signingAlgo string) (crypto.Signer, error)
```

### Func PublicKey
```go
func PublicKey(ctx context.Context, client Client, keyID string) (crypto.PublicKey, error)
```



## Types
### Type Client
```go
type Client interface {
	Sign(ctx context.Context, input *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, input *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
}
```


### Type Signer
```go
type Signer struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (s *Signer) Public() crypto.PublicKey
```


```go
func (s *Signer) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error)
```







