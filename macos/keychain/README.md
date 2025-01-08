# Package [cloudeng.io/macos/keychain](https://pkg.go.dev/cloudeng.io/macos/keychain?tab=doc)

```go
import cloudeng.io/macos/keychain
```

Package keychain provides support for working with the macos keychain.

## Functions
### Func ReadSecureNote
```go
func ReadSecureNote(account, service string) ([]byte, error)
```
ReadSecureNote reads a secure note from a local, non-icloud, keychain.
The note may be in plist format if it was created directly in the keychain
using keychain access.

### Func WriteSecureNote
```go
func WriteSecureNote(account, service string, data []byte) error
```
WriteSecureNote writes a secure note to a local, non-icloud, keychain.




