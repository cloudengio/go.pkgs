# Package [cloudeng.io/security/keys/keychain/plugins](https://pkg.go.dev/cloudeng.io/security/keys/keychain/plugins?tab=doc)

```go
import cloudeng.io/security/keys/keychain/plugins
```


## Functions
### Func Plugin
```go
func Plugin(in io.Reader, out io.Writer) error
```
Plugin is the entry point for the macOS keychain plugin. It reads a Request
from in and writes a Response to out.



## Types
### Type Request
```go
type Request struct {
	Account  string `json:"account"`
	Keyname  string `json:"keyname"`
	WriteKey bool   `json:"write_key,omitempty"` // if true, write the key
	Contents string `json:"contents,omitempty"`  // base64 encoded contents for writing
}
```
Request represents the request to the keychain plugin.


### Type Response
```go
type Response struct {
	Account  string `json:"account"`
	Keyname  string `json:"keyname"`
	Contents string `json:"contents"` // base64 encoded
	Error    string `json:"error,omitempty"`
}
```
Response represents the response from the keychain plugin.





