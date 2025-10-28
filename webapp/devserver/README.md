# Package [cloudeng.io/webapp/devserver](https://pkg.go.dev/cloudeng.io/webapp/devserver?tab=doc)

```go
import cloudeng.io/webapp/devserver
```


## Types
### Type DevServer
```go
type DevServer struct {
	// contains filtered or unexported fields
}
```
DevServer represents a development server, such as provided by webpack or
vite that serves UI content with live reload capabilities.

### Functions

```go
func NewServer(ctx context.Context, dir, binary string, args ...string) *DevServer
```
NewServer creates a new DevServer instance that will manage the lifecycle
of the supplied exec.Cmd instance. The stdout of the command is scanned
line-by-line and passed to the supplied URLExtractor function until a URL is
successfully extracted.



### Methods

```go
func (ds *DevServer) Close()
```
CloseStdout closes the stdout from the dev server process and will prevent
any further output from being processed or forwarded to the writer supplied
to StartAndWaitForURL.


```go
func (ds *DevServer) StartAndWaitForURL(ctx context.Context, writer io.Writer, extractor URLExtractor) (*url.URL, error)
```
StartAndWaitForURL starts the dev server and waits until a URL is extracted
from its output using the supplied URLExtractor function. The context can be
used to cancel the wait operation. If the context is cancelled before a URL
is extracted an error is returned.




### Type URLExtractor
```go
type URLExtractor func(line []byte) (*url.URL, error)
```
URLExtractor parses each line of output from the dev server looking for a
URL to which requests can be proxied. If a URL is successfully extracted it
is returned with a nil error. If the line does not contain a URL, then a nil
URL and a nil error are returned. If the line should contain a URL but it
cannot be extracted then a nil URL and a non-nil error should be returned.

### Functions

```go
func NewViteURLExtractor(re *regexp.Regexp) URLExtractor
```
NewViteURLExtractor returns a URLExtractor that extracts the URL from lines
that match the supplied regexp. If re is nil a default regexp that matches
lines starting with "➜ Local:" is used. Example matching lines:

    ➜  Local:   http://localhost:5173/

Vite output typically looks as follows:

    > webapp-sample-vite@0.0.0 dev
    > vite --host

      ROLLDOWN-VITE v7.1.14  ready in 71 ms

      ➜  Local:   http://localhost:5173/
      ➜  Network: http://172.16.1.222:5173/
      ➜  Network: http://172.16.1.142:5173/


```go
func NewWebpackURLExtractor(re *regexp.Regexp) URLExtractor
```
NewWebpackURLExtractor returns a URLExtractor that extracts the URL from
lines that match the supplied regexp. If re is nil a default regexp that
matches lines containing "Local:" is used. Example matching lines:

    Local:     http://localhost:8080/

Webpack output typically looks as follows:

Compiled successfully!

You can now view webapp-sample in the browser.

    Local:            http://localhost:3000
    On Your Network:  http://172.16.1.222:3000

Note that the development build is not optimized. To create a production
build, use npm run build.

webpack compiled successfully







