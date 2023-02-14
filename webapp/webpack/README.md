# Package [cloudeng.io/webapp/webpack](https://pkg.go.dev/cloudeng.io/webapp/webpack?tab=doc)

```go
import cloudeng.io/webapp/webpack
```


## Types
### Type DevServer
```go
type DevServer struct {
	// contains filtered or unexported fields
}
```
DevServer represents a webpack dev server.

### Functions

```go
func NewDevServer(ctx context.Context, dir string, command string, args ...string) *DevServer
```
NewDevServer creates a new instance of DevServer. Note, that the server
is not started at this point. The dir argument specifies the directory
containing the webpack configuration. Context, command and args are passed
to exec.CommandContext. A typical usage would be:

    NewDevServer(ctx, "./frontend", "webpack", "serve", "-c", "webpack.dev.js")

Additional, optional configuration is possible via the Configure method.



### Methods

```go
func (ds *DevServer) Configure(opts ...DevServerOption)
```
Configure applies options and mus be called before Start.


```go
func (ds *DevServer) Shutdown()
```
Shutdown asks the dev server to shut itself down.


```go
func (ds *DevServer) Start() error
```
Start starts the dev server.


```go
func (ds *DevServer) WaitForURL(ctx context.Context) (*url.URL, error)
```
WaitForURL parses the output of the development server looking for a line
that specifies the URL it is listening on.




### Type DevServerFlags
```go
type DevServerFlags struct {
	WebpackDir    string `subcmd:"webpack-dir,,'set to a directory containing a webpack configuration with the webpack dev server configured. This dev server will then be started and requests proxied to it.'"`
	WebpackServer string `subcmd:"webpack-server,,set to the url of an already running webpack dev server to which requests will be proxied."`
}
```
DevServerFlags represents the flags commonly used when using webpack dev
servers.


### Type DevServerOption
```go
type DevServerOption func(ds *DevServer)
```
DevServerOption represents an option to Configure.

### Functions

```go
func AddrRegularExpression(re *regexp.Regexp) DevServerOption
```
AddrRegularExpression specifies the regular expression to use for
determining the running server's address. The default RE is: 'Project is
running at'. However, some configurations may use a different output.


```go
func SetSdoutStderr(stdout, stderr io.Writer) DevServerOption
```
SetSdoutStderr sets the stdout and stderr io.Writers to be used by the dev
server.







