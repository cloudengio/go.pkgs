# Package [cloudeng.io/cmdutil/keys/keyscmd](https://pkg.go.dev/cloudeng.io/cmdutil/keys/keyscmd?tab=doc)

```go
import cloudeng.io/cmdutil/keys/keyscmd
```

Package keyscmd provides a set of utilities for reading and writing multiple
keys stored in a single item in a file system using the format used by
keys.InMemoryKeyStore.

## Constants
### KeysSubcmdTree
```go
KeysSubcmdTree = `
- name: key-info
  summary: manage key info items in a keychain/secrets store, multiple key info items can be stored in a single item. In all cases if input or output is a filename, then "-" or "" will result in stdin or stdout being used as appropriate.
  commands:
    - name: create
      summary: create a new key info, including secret, and write it to <filename>
      arguments:
        - <filename>
    - name: list
      summary: list all key info items in an item
    - name: get
      summary: get a key info from an item from the keychain and write it to <filename>
      arguments:
        - <filename>
    - name: set
      summary: set a key info, read from filename, in an item in the keychain. If the key info already exists it will be overwritten.
      arguments:
        - <filename>
    - name: delete
      summary: delete a key info from an item in the keychain
`

```



## Variables
### ErrKeyInfoNotFound
```go
ErrKeyInfoNotFound = errors.New("key info not found")

```

### ErrUpdateNotAllowed
```go
ErrUpdateNotAllowed = errors.New("update not allowed")

```



## Functions
### Func CopyContents
```go
func CopyContents(ctx context.Context, srcFS file.ReadFileFS, src string, dstFS file.WriteFileFS, dst string, perm fs.FileMode) error
```
CopyContents copies the contents of the source file in srcFS to the
destination file in dstFS.

### Func IsDstSafe
```go
func IsDstSafe(dst string) error
```
SafeStdout checks if the provided name refers to Stdout ("-" or empty) and
if so, verifies that Stdout is piped. If Stdout is not piped, it returns an
error.

### Func IsStdoutStdin
```go
func IsStdoutStdin(name string) bool
```
IsStdoutStdin returns true if the provided name is "-" or empty, indicating
that the operation should read from stdin or write to stdout.

### Func NewKeyInfoExtenstion
```go
func NewKeyInfoExtenstion(name string, appendFn func(cmd *subcmd.CommandSetYAML) error) subcmd.Extension
```

### Func ReadFSWithStdin
```go
func ReadFSWithStdin(fs file.ReadFileFS, name string) file.ReadFileFS
```
ReadFSWithStdin returns a file.ReadFileFS that reads from stdin if the name
is "-" or empty.

### Func ReadFromLocal
```go
func ReadFromLocal(ctx context.Context, filename string, dstFS file.WriteFileFS, dst string, perm fs.FileMode) error
```

### Func ReadWriteFSWithStdout
```go
func ReadWriteFSWithStdout(fs file.ReadWriteFileFS, name string) file.ReadWriteFileFS
```
ReadWriterFSWithStdout returns a file.ReadWriteFileFS that reads from fs and
writes to stdout if the name is "-" or empty.

### Func SafeWriteToLocal
```go
func SafeWriteToLocal(ctx context.Context, srcFS file.ReadFileFS, src string, filename string, perm fs.FileMode) error
```

### Func WriteFSWithStdout
```go
func WriteFSWithStdout(fs file.WriteFileFS, name string) file.WriteFileFS
```
WriterSWithStdout returns a file.WriteFileFS that writes to stdout if the
name is "-" or empty.



## Types
### Type KeyReader
```go
type KeyReader struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewKeyReader(fs file.ReadFileFS) KeyReader
```
NewKeyReader creates a new KeyReader that reads keys stored using the
InMemoryKeyStore format using the provided file.ReadFileFS.



### Methods

```go
func (r *KeyReader) GetKey(ctx context.Context, name string, spec keys.KeySpec) (keys.Info, error)
```
GetKey retrieves a specific key from the specified item in the file system
based on the provided keys.KeySpec. If the key is not found, it returns an
error.


```go
func (r *KeyReader) GetKeys(ctx context.Context, name string) ([]keys.Info, error)
```
GetKeys reads all keys from the specified item in the file system and
returns them as a slice of keys.Info.


```go
func (r *KeyReader) SafeWriteKeyInfoJSON(ctx context.Context, ki keys.Info, dst string, perm fs.FileMode) error
```


```go
func (r *KeyReader) SafeWriteKeyInfoToLocal(ctx context.Context, ki keys.Info, marshal func(any) ([]byte, error), dst string, perm fs.FileMode) error
```


```go
func (r *KeyReader) SafeWriteKeyInfoYAML(ctx context.Context, ki keys.Info, dst string, perm fs.FileMode) error
```




### Type KeySpecFlags
```go
type KeySpecFlags struct {
	ID   string `subcmd:"key-id,,key id"`
	User string `subcmd:"key-user,,key user"`
}
```

### Methods

```go
func (f KeySpecFlags) KeySpec() keys.KeySpec
```




### Type KeyWriter
```go
type KeyWriter struct {
	KeyReader
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewKeyWriter(fs file.ReadWriteFileFS) KeyWriter
```
NewKeyWriter creates a new KeyWriter that writes keys to an InMemoryKeyStore
using the provided file.ReadWriteFileFS in the InMemoryKeyStore format.



### Methods

```go
func (w *KeyWriter) DeleteKey(ctx context.Context, name string, spec keys.KeySpec) error
```
DeleteKey removes a specific key from the specified item in the file system
based on the provided keys.KeySpec. It works by reading all of the existing
keys, removing the specified key, and then writing the updated list back to
the file system.


```go
func (w *KeyWriter) ReadKeyInfoFromLocal(ctx context.Context, filename string, unmarshal func([]byte, any) error) (keys.Info, error)
```


```go
func (w *KeyWriter) ReadKeyInfoFromLocalJSON(ctx context.Context, filename string) (keys.Info, error)
```


```go
func (w *KeyWriter) ReadKeyInfoFromLocalYAML(ctx context.Context, filename string) (keys.Info, error)
```


```go
func (w *KeyWriter) SetKeys(ctx context.Context, name string, update bool, keys ...keys.Info) error
```
SetKeys adds or updates keys in the specified item in the file system.
If update is false, it will return an error if any of the keys already exist
in the item. If update is true, it will overwrite existing keys with the
same user and ID.




### Type SecretConfig
```go
type SecretConfig struct {
	Size   int          `yaml:"key-size" doc:"size of the secret in bytes"`
	Format SecretFormat `yaml:"key-format" doc:"format of the secret, one of raw, hex, base64"`
	ID     string       `yaml:"key-id" doc:"id of the key"`
	User   string       `yaml:"key-user" doc:"user/owner associated with the key"`
}
```

### Methods

```go
func (sc SecretConfig) New() (keys.Info, error)
```
NewSecret generates a new random secret of the specified size in bytes and
format and returns it as a keys.Info object.




### Type SecretConfigFlags
```go
type SecretConfigFlags struct {
	Size   int                      `subcmd:"size,32,size of the secret in bytes"`
	Format flags.Enum[SecretFormat] `subcmd:"format,hex,'format of the secret, one of raw, hex, base64'"`
	ID     string                   `subcmd:"id,,id of the key"`
	User   string                   `subcmd:"user,,user/owner associated with the key"`
}
```

### Methods

```go
func (sf SecretConfigFlags) SecretConfig() SecretConfig
```




### Type SecretFormat
```go
type SecretFormat int
```
SecretFormat represents the format in which the secret is represented,
such as raw bytes, hexadecimal, or base64 encoding.

### Constants
### SecretFormatRaw, SecretFormatHex, SecretFormatBase64
```go
// SecretFormatRaw indicates that the secret is stored in raw byte format.
SecretFormatRaw SecretFormat = iota
// SecretFormatHex indicates that the secret is stored in hexadecimal format.
SecretFormatHex
// SecretFormatBase64 indicates that the secret is stored in base64.StdEncoding format.
SecretFormatBase64

```



### Methods

```go
func (f SecretFormat) EnumValues() map[string]SecretFormat
```
SecretFormat implements the flags.Enum interface for SecretFormat.







