# Package [cloudeng.io/security/keys/keychain](https://pkg.go.dev/cloudeng.io/security/keys/keychain?tab=doc)

```go
import cloudeng.io/security/keys/keychain
```

Package keychain provides functionality to interact with a local keychain.
It is intended to be used to retrieve a secret that's used to
encrypt/decrypt all other api tokens. A plugin is used to avoid the need
to reauthenticate with the OS every time an application that needs the key
is recompiled during development. For production applications the keychain
should be accessed directly via the Plugin method.

## Constants
### KeyChainPluginName
```go
KeyChainPluginName = "keychain_plugin_cmd"

```



## Functions
### Func ExtPluginBuildCommand
```go
func ExtPluginBuildCommand(ctx context.Context) *exec.Cmd
```
DevelopmentPluginBuildCommand returns an exec.Cmd that builds the keychain
plugin for the current OS and installs it as <dir>/<name>. If name is empty,
it defaults to DefaultExtPluginName. The returned command can be executed
to build the plugin, and the second return value is the location where the
plugin will be installed.

### Func GetKey
```go
func GetKey(ctx context.Context, account, keyname string) ([]byte, error)
```
GetKey retrieves a key from the keychain using the specified plugin
(extPluginPath) if the application is running via `go run`, or directly if
it is a compiled binary.

### Func IsExtPluginAvailable
```go
func IsExtPluginAvailable(ctx context.Context) bool
```
IsExtPluginAvailable checks if the external keychain plugin is available.

### Func RunAvailablePlugin
```go
func RunAvailablePlugin(ctx context.Context, req plugins.Request) (plugins.Response, error)
```
RunAvailablePlugin decides whether to use the external plugin or the
compiled-in plugin based on whether the application is running via `go run`.

### Func RunExtPlugin
```go
func RunExtPlugin(ctx context.Context, binary string, req plugins.Request) (plugins.Response, error)
```
RunExtPlugin runs an external keychain plugin with the provided request
and returns the response. binary is either a command on the PATH or an
absolute path to the plugin executable. If binary is empty it defaults to
KeyChainPluginName. The default external plugin can be installed using the
WithExternalPlugin function.

### Func RunPlugin
```go
func RunPlugin(ctx context.Context, req plugins.Request) (plugins.Response, error)
```
RunPlugin executes the keychain plugin compiled into the running
application.

### Func SetKey
```go
func SetKey(ctx context.Context, account, keyname string, contents []byte) error
```
SetKey sets a key in the keychain using the specified plugin
(extPluginPath). If the application is running via `go run`, it uses the
external plugin; otherwise, it uses the compiled-in plugin. The key is
written as base64 encoded contents. If the key already exists, it will be
overwritten. The account can be empty, in which case the default account is
used. The keyname must not be empty.

### Func WithExternalPlugin
```go
func WithExternalPlugin(ctx context.Context, extPluginPath string) error
```
WithExternalPlugin builds the external plugin if the application is running
via `go run`. It uses the ExtPluginBuildCommand to build the plugin. If
the build fails, it returns an error with the output of the build command.
If the application is not running via `go run`, it does nothing and returns
nil. This function is intended to be called at the start of the application
to ensure that the external plugin is built and available for use.



## Types
### Type KeyChainReadFS
```go
type KeyChainReadFS interface {
	ReadFileCtx(ctx context.Context, name string) ([]byte, error)
}
```
KeyChainReadFS defines an interface for reading files from a keychain via an
internal or external plugin.


### Type KeyChainWriteFS
```go
type KeyChainWriteFS interface {
	WriteFileCtx(ctx context.Context, name string, data []byte) error
}
```
KeyChainWriteFS defines an interface for writing files to a keychain via an
internal or external plugin.


### Type PluginFS
```go
type PluginFS struct {
	// contains filtered or unexported fields
}
```
PluginFS combines both reading and writing capabilities for a keychain via
an internal or external plugin.

### Functions

```go
func NewPluginFS(account string) *PluginFS
```
NewPluginFS creates a new PluginFS instance with the specified account.
An external plugin, if installed, will be used when process is run via 'go
run', but not when run as a pre-compiled binary.



### Methods

```go
func (p *PluginFS) ReadFileCtx(ctx context.Context, name string) ([]byte, error)
```


```go
func (p *PluginFS) WriteFileCtx(ctx context.Context, name string, data []byte) error
```







