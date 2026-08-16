# Package [cloudeng.io/security/keys/keychain/keychaintestutil](https://pkg.go.dev/cloudeng.io/security/keys/keychain/keychaintestutil?tab=doc)

```go
import cloudeng.io/security/keys/keychain/keychaintestutil
```

Package keychaintestutil provides test utilities for keychain plugins,
including an in-memory implementation of the keychain plugin protocol.

## Constants
### SocketEnvVar
```go
// SocketEnvVar is the environment variable that child processes can check
// for the daemon socket path or address.
SocketEnvVar = "KEYCHAIN_TEST_SOCKET"

```



## Functions
### Func BuildPluginBinary
```go
func BuildPluginBinary(t testing.TB) string
```
BuildPluginBinary compiles the in-memory test plugin executable into a
temporary directory and registers its cleanup with t.Cleanup. It returns the
absolute path to the binary.

### Func Main
```go
func Main()
```
Main is the main entry point for a keychain test plugin executable.

### Func Run
```go
func Run(ctx context.Context, r io.Reader, w io.Writer, stderr io.Writer, args ...string) error
```
Run executes the test plugin CLI logic reading from r and writing to w.
It parses args, connects to a daemon if specified via flag or env var,
or handles the request locally.



## Types
### Type FS
```go
type FS struct {
	// contains filtered or unexported fields
}
```
FS is an in-memory filesystem that reads and writes keys directly from
an in-memory Plugin without running external processes. It implements
file.ReadFileFS and file.WriteFileFS.

### Functions

```go
func NewFS(plugin *Plugin, writable bool) *FS
```
NewFS creates a new in-memory FS instance backed by the given Plugin.



### Methods

```go
func (f *FS) Plugin() *Plugin
```
Plugin returns the underlying Plugin.


```go
func (f *FS) ReadFile(name string) ([]byte, error)
```
ReadFile reads a key from the in-memory store.


```go
func (f *FS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```
ReadFileCtx reads a key from the in-memory store with context.


```go
func (f *FS) WriteFile(name string, data []byte, perm fs.FileMode) error
```
WriteFile writes a key to the in-memory store.


```go
func (f *FS) WriteFileCtx(ctx context.Context, name string, data []byte, _ fs.FileMode) error
```
WriteFileCtx writes a key to the in-memory store with context.




### Type Plugin
```go
type Plugin struct {
	// contains filtered or unexported fields
}
```
Plugin is an in-memory implementation of a keychain plugin. It stores keys
in memory and implements the plugin request/response protocol.

### Functions

```go
func New() *Plugin
```
New creates a new in-memory Plugin.



### Methods

```go
func (p *Plugin) Clear()
```
Clear removes all keys and configured errors from memory.


```go
func (p *Plugin) Delete(keyname string)
```
Delete removes a key from memory.


```go
func (p *Plugin) Get(keyname string) ([]byte, bool)
```
Get retrieves the contents of a key from memory.


```go
func (p *Plugin) HandleRequest(_ context.Context, req plugins.Request) plugins.Response
```
HandleRequest processes a single plugins.Request and returns the
corresponding plugins.Response.


```go
func (p *Plugin) Keys() []string
```
Keys returns a slice of all key names currently in memory.


```go
func (p *Plugin) ServeIO(ctx context.Context, r io.Reader, w io.Writer) error
```
ServeIO reads a JSON-encoded Request from r, handles it with HandleRequest,
and writes the JSON-encoded Response to w.


```go
func (p *Plugin) Set(keyname string, contents []byte)
```
Set stores a key and its contents in memory.


```go
func (p *Plugin) SetDefaultError(err *plugins.Error)
```
SetDefaultError configures a default error to return for all requests.


```go
func (p *Plugin) SetError(keyname string, err *plugins.Error)
```
SetError configures a specific error to return for operations on the given
keyname.




### Type Server
```go
type Server struct {
	// contains filtered or unexported fields
}
```
Server is a local daemon that serves an in-memory Plugin over a Unix domain
socket (or local TCP on systems without Unix domain sockets).

### Functions

```go
func StartServer(ctx context.Context, plugin *Plugin, socketPath ...string) (*Server, error)
```
StartServer starts a local Server backed by the specified in-memory Plugin.
If socketPath is empty, a temporary socket path is created.



### Methods

```go
func (s *Server) Address() string
```
Address returns the socket path or address of the Server.


```go
func (s *Server) Close() error
```
Close stops the server and cleans up the socket file.


```go
func (s *Server) Network() string
```
Network returns the network type ("unix" or "tcp").


```go
func (s *Server) Plugin() *Plugin
```
Plugin returns the underlying in-memory Plugin.







