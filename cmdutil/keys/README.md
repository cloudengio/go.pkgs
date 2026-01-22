# Package [cloudeng.io/cmdutil/keys](https://pkg.go.dev/cloudeng.io/cmdutil/keys?tab=doc)

```go
import cloudeng.io/cmdutil/keys
```

Package keys provides types and utilities for managing API keys/tokens.
A key consists of an identifier, an optional user, a token value, and
optional extra information. The package includes an in-memory key store for
storing and retrieving keys, as well as context utilities for passing key
stores across API boundaries.

## Functions
### Func ContextWithKey
```go
func ContextWithKey(ctx context.Context, ki Info) context.Context
```
ContextWithKey returns a new context with the provided KeyInfo added to an
InMemoryKeyStore. If no InMemoryKeyStore exists in the context, a new one is
created.

### Func ContextWithKeyStore
```go
func ContextWithKeyStore(ctx context.Context, ims *InMemoryKeyStore) context.Context
```
ContextWithKeyStore returns a new context with the provided
InMemoryKeyStore.

### Func ContextWithoutKeyStore
```go
func ContextWithoutKeyStore(ctx context.Context) context.Context
```
ContextWithoutKeyStore returns a new context without an InMemoryKeyStore.



## Types
### Type InMemoryKeyStore
```go
type InMemoryKeyStore struct {
	// contains filtered or unexported fields
}
```
InMemoryKeyStore is a simple in-memory key store intended for passing a
small number of keys within an application. It will typically be stored in a
context.Context to ease passing it across API boundaries.

### Functions

```go
func KeyStoreFromContext(ctx context.Context) (*InMemoryKeyStore, bool)
```
KeyStoreFromContext retrieves the InMemoryKeyStore from the context.


```go
func NewInMemoryKeyStore() *InMemoryKeyStore
```
NewInMemoryKeyStore creates a new InMemoryKeyStore instance.



### Methods

```go
func (ims *InMemoryKeyStore) Add(key Info)
```


```go
func (ims *InMemoryKeyStore) Get(id string) (Info, bool)
```
Get retrieves a key by its ID. It returns the key and a boolean indicating
whether the key was found.


```go
func (ims *InMemoryKeyStore) KeyOwners() []KeyOwner
```
KeyOwners returns the owners of keys in the store.


```go
func (ims *InMemoryKeyStore) Len() int
```


```go
func (ims *InMemoryKeyStore) MarshalJSON() ([]byte, error)
```
MarshalJSON implements the json.Marshaler interface to allow marshaling the
InMemoryKeyStore to JSON.


```go
func (ims *InMemoryKeyStore) MarshalYAML() (any, error)
```
MarshalYAML implements the yaml.Marshaler interface to allow marshaling the
InMemoryKeyStore to YAML.


```go
func (ims *InMemoryKeyStore) ReadJSON(ctx context.Context, fs file.ReadFileFS, name string) error
```
ReadJSON reads key information from a JSON file using the provided
file.ReadFileFS and unmarshals it into the InMemoryKeyStore.


```go
func (ims *InMemoryKeyStore) ReadYAML(ctx context.Context, fs file.ReadFileFS, name string) error
```
ReadYAML reads key information from a YAML file using the provided
file.ReadFileFS and unmarshals it into the InMemoryKeyStore.


```go
func (ims *InMemoryKeyStore) UnmarshalJSON(data []byte) error
```
UnmarshalJSON implements the json.Unmarshaler interface to allow
unmarshaling from both a list and a map of keys. textutil.TrimUnicodeQuotes
is used on the ID, User, and Token fields.


```go
func (ims *InMemoryKeyStore) UnmarshalYAML(node *yaml.Node) error
```
UnmarshalYAML implements the yaml.Unmarshaler interface to allow
unmarshaling from both a list and a map of keys. textutil.TrimUnicodeQuotes
is used on the ID, User, and Token fields.




### Type Info
```go
type Info struct {
	ID   string
	User string
	// contains filtered or unexported fields
}
```
Info represents a specific key and associated information and is intended
to be reused and referred to by it's ID. It can be parsed from json or yaml
representations with the following fields:
  - key_id: the identifier for the key
  - user: optional user associated with the key
  - token: the token value
  - extra: optional extra information as a json or yaml object

In addition, extra information can be set directly using WithExtra and
retrieved using UnmarshalExtra. If WithExtra is called and the KeyInfo
instance has already been unmarshaled from json or yaml then the extra
information from unmarshaling will be deleted and the new extra information
will be stored in its place.

An Info instance can be created/populated using NewInfo or by unmarshaling
from json or yaml.

### Functions

```go
func KeyInfoFromContextForID(ctx context.Context, id string) (Info, bool)
```
KeyInfoFromContextForID retrieves the KeyInfo for the specified ID from the
context.


```go
func NewInfo(id, user string, token []byte) Info
```
NewInfo creates a new Info instance with the specified id, user, token.
The token slice is cloned and the input slice is zeroed. Extra information
can be set using WithExtra and accessed using UnmarshalExtra.



### Methods

```go
func (k Info) GetExtra() any
```
GetExtra returns the extra information for the key.


```go
func (k Info) MarshalJSON() ([]byte, error)
```


```go
func (k Info) MarshalYAML() (any, error)
```


```go
func (k Info) String() string
```
String returns a string representation of the KeyInfo with the Token and
Extra fields redacted.


```go
func (k Info) Token() *Token
```


```go
func (k Info) UnmarshalExtra(v any) error
```
UnmarshalExtra unmarshals the extra json, yaml, or explicitly stored extra
information into the provided value. It does not modify the stored extra
information. If the extra information is stored as a json.RawMessage then
it will be unmarshaled into the provided value. If the extra information is
stored as a yaml.Node then it will be decoded into the provided value. If
the extra information was provided using WithExtra then it will be assigned
to the supplied value v, provided that v is a pointer to a type to which the
extra information is assignable.


```go
func (k *Info) UnmarshalJSON(data []byte) error
```
UnmarshalJSON implements the json.Unmarshaler interface and calls
textutil.TrimUnicodeQuotes on the ID, User, and Token fields.


```go
func (k *Info) UnmarshalYAML(node *yaml.Node) error
```
UnmarshalYAML implements the yaml.Unmarshaler interface and calls
textutil.TrimUnicodeQuotes on the ID, User, and Token fields.


```go
func (k *Info) WithExtra(v any)
```
WithExtra sets the extra information for the key. Extra information can be
accessed using UnmarshalExtra or GetExtra. WithExtra is intended to be used
to associate extra information that is not obtained via unmarshaling from
json or yaml.




### Type KeyOwner
```go
type KeyOwner struct {
	ID   string
	User string
}
```
KeyOwner represents the owner of a key, identified by an ID and an optional
user.

### Methods

```go
func (ko KeyOwner) String() string
```




### Type Token
```go
type Token struct {
	KeyOwner
	// contains filtered or unexported fields
}
```
Token represents an API token. It is intended for temporary use with the
Clear() method being called to zero the token value when it is no longer
needed, typically using a defer statement. It consists of an ID and a token
value with the ID purely for identification purposes.

### Functions

```go
func NewToken(id, user string, value []byte) Token
```
NewToken creates a new Token instance, cloning the provided value and
zeroing the input slice.


```go
func TokenFromContextForID(ctx context.Context, id string) (*Token, bool)
```
TokenFromContextForID retrieves the Token for the specified ID from the
context.



### Methods

```go
func (t *Token) Clear()
```
Clear zeros the token value.


```go
func (t Token) String() string
```


```go
func (t Token) Value() []byte
```
Value returns the value of the token.







