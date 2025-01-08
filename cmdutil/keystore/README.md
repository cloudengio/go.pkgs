# Package [cloudeng.io/cmdutil/keystore](https://pkg.go.dev/cloudeng.io/cmdutil/keystore?tab=doc)

```go
import cloudeng.io/cmdutil/keystore
```


## Functions
### Func ContextWithAuth
```go
func ContextWithAuth(ctx context.Context, am Keys) context.Context
```



## Types
### Type KeyInfo
```go
type KeyInfo struct {
	ID    string `yaml:"key_id"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
}
```
KeyInfo represents a specific key configuration and is intended to be reused
and referred to by it's key_id.

### Functions

```go
func AuthFromContextForID(ctx context.Context, id string) KeyInfo
```



### Methods

```go
func (k KeyInfo) String() string
```




### Type Keys
```go
type Keys map[string]KeyInfo
```
Keys is a map of ID/key_id to KeyInfo

### Functions

```go
func Parse(data []byte) (Keys, error)
```
Parse parses the supplied data into an AuthInfo map.


```go
func ParseConfigFile(ctx context.Context, filename string) (Keys, error)
```
ParseConfigFile calls cmdyaml.ParseConfigFile for Keys.


```go
func ParseConfigURI(ctx context.Context, filename string, handlers map[string]cmdyaml.URLHandler) (Keys, error)
```
ParseConfigURI calls cmdyaml.ParseConfigURI for Keys.







