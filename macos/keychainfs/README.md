# Package [cloudeng.io/macos/keychainfs](https://pkg.go.dev/cloudeng.io/macos/keychainfs?tab=doc)

```go
import cloudeng.io/macos/keychainfs
```


## Functions
### Func DefaultAccount
```go
func DefaultAccount() string
```
DefaultAccount returns the current user's account name.

### Func NewSecureNoteFSFromURL
```go
func NewSecureNoteFSFromURL(ctx context.Context, u *url.URL) (nctx context.Context, notename string)
```
NewSecureNoteFSFromURL creates a new context containing a new SecureNoteFS
based on the supplied URL and the name of the note within the keychain.
The URL should be of the form keychain:///note?account=accountname



## Types
### Type Option
```go
type Option func(*options)
```
Option provides options for configuring a SecureNoteFS.

### Functions

```go
func WithAccount(account string) Option
```
WithAccount specifies the account name to use with New.




### Type SecureNoteFS
```go
type SecureNoteFS struct {
	// contains filtered or unexported fields
}
```
SecureNoteFS implements an fs.ReadFS that reads secure notes from the macOS
keychain.

### Functions

```go
func NewSecureNoteFS(opts ...Option) *SecureNoteFS
```
NewSecureNoteFS creates a new SecureNoteFS.



### Methods

```go
func (fs *SecureNoteFS) Open(name string) (fs.File, error)
```


```go
func (fs *SecureNoteFS) ReadFile(name string) ([]byte, error)
```


```go
func (fs *SecureNoteFS) WriteFile(name string, data []byte, _ fs.FileMode) error
```


```go
func (fs *SecureNoteFS) WriteFileCtx(_ context.Context, name string, data []byte, _ fs.FileMode) error
```







