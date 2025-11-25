# Package [cloudeng.io/cmdutil/keys](https://pkg.go.dev/cloudeng.io/cmdutil/keys?tab=doc)

```go
import cloudeng.io/cmdutil/keys
```


## Functions
### Func ContextWithAuth
```go
func ContextWithAuth(ctx context.Context, ims InmemoryKeyStore) context.Context
```



## Types
### Type InmemoryKeyStore
```go
type InmemoryKeyStore struct {
	// contains filtered or unexported fields
}
```
InmemoryKeyStore is a simple in-memory key store intended for passing a
small number of keys within an application. It will typically be stored in a
context.Context to ease passing it across API boundaries.

### Functions

```go
func NewInmemoryKeyStore() *InmemoryKeyStore
```
NewInmemoryKeyStore creates a new InmemoryKeyStore instance.



### Methods

```go
func (ims *InmemoryKeyStore) AddKey(key KeyInfo)
```


```go
func (ims *InmemoryKeyStore) GetAllKeys() []KeyInfo
```
GetAllKeys returns all keys in the store.


```go
func (ims *InmemoryKeyStore) GetKey(id string) (KeyInfo, bool)
```
GetKey retrieves a key by its ID. It returns the key and a boolean
indicating whether the key was found.


```go
func (ims *InmemoryKeyStore) UnmarshalYAML(node *yaml.Node) error
```
UnmarshalYAML implements the yaml.Unmarshaler interface to allow
unmarshaling from both a list and a map of keys.




### Type KeyInfo
```go
type KeyInfo struct {
	ID    string `yaml:"key_id"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
	Extra any    `yaml:"extra,omitempty"` // Extra can be used to store additional information about the key
}
```
KeyInfo represents a specific key and associated information and is intended
to be reused and referred to by it's key_id.

### Functions

```go
func AuthFromContextForID(ctx context.Context, id string) (KeyInfo, bool)
```



### Methods

```go
func (k KeyInfo) ExtraAs(v any) error
```
ExtraAs attempts to unmarshal the Extra field into the provided struct.


```go
func (k KeyInfo) String() string
```







