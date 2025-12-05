# Package [cloudeng.io/cmdutil/keys/unsafekeystore](https://pkg.go.dev/cloudeng.io/cmdutil/keys/unsafekeystore?tab=doc)

```go
import cloudeng.io/cmdutil/keys/unsafekeystore
```

Package unsafekeystore is intended to document the use of plaintext,
local filesystems being used to store keys.

## Functions
### Func New
```go
func New() file.ReadFileFS
```
New returns a new instance of an unsafekeystore that reads keys from a
plaintext, local file.




