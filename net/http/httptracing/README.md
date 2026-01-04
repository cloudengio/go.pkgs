# Package [cloudeng.io/net/http/httptracing](https://pkg.go.dev/cloudeng.io/net/http/httptracing?tab=doc)

```go
import cloudeng.io/net/http/httptracing
```


## Functions
### Func JSONHandlerRequestLogger
```go
func JSONHandlerRequestLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte)
```
JSONHandlerRequestLogger logs the request body as a JSON object. The
supplied logger is pre-configured with relevant request information.

### Func JSONHandlerResponseLogger
```go
func JSONHandlerResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ http.Header, statusCode int, data []byte)
```
JSONHandlerResponseLogger logs the response body from an http.Handler as a
JSON object.

### Func JSONOrTextHandlerResponseLogger
```go
func JSONOrTextHandlerResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ http.Header, statusCode int, data []byte)
```
JSONOrTextHandlerResponseLogger logs the response body from an http.Handler
as a JSON object if it is valid JSON, otherwise as text. Use the JSON or
Text variants wherever possible as they are more efficient.

### Func JSONOrTextRequestLogger
```go
func JSONOrTextRequestLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte)
```
JSONOrTextRequestLogger logs the request body as a JSON object if it is
valid JSON, otherwise as text. Use the JSON or Text variants wherever
possible as they are more efficient. The supplied logger is pre-configured
with relevant request information.

### Func JSONOrTextResponseLogger
```go
func JSONOrTextResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ *http.Response, data []byte)
```
JSONOrTextResponseLogger logs the response body as a JSON object if it
is valid JSON, otherwise as text. Use the JSON or Text variants wherever
possible as they are more efficient. The supplied logger is pre-configured
with relevant request information.

### Func JSONRequestLogger
```go
func JSONRequestLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte)
```
JSONRequestLogger logs the request body as a JSON object. The supplied
logger is pre-configured with relevant request information.

### Func JSONResponseLogger
```go
func JSONResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ *http.Response, data []byte)
```
JSONResponseLogger logs the response body as a JSON object. The supplied
logger is pre-configured with relevant request information.

### Func TextHandlerRequestLogger
```go
func TextHandlerRequestLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte)
```
TextHandlerRequestLogger logs the request body as a text object. The
supplied logger is pre-configured with relevant request information.

### Func TextHandlerResponseLogger
```go
func TextHandlerResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ http.Header, statusCode int, data []byte)
```
TextHandlerResponseLogger logs the response body from an http.Handler as a
text object.

### Func TextRequestLogger
```go
func TextRequestLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte)
```
TextRequestLogger logs the request body as a text object. The supplied
logger is pre-configured with relevant request information.

### Func TextResponseLogger
```go
func TextResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ *http.Response, data []byte)
```
TextResponseLogger logs the response body as a text object. The supplied
logger is pre-configured with relevant request information.



## Types
### Type TraceHandlerOption
```go
type TraceHandlerOption func(*handlerOptions)
```
TraceHandlerOption is the type for options that can be passed to
NewTracingHandler.

### Functions

```go
func WithHandlerLogger(logger *slog.Logger) TraceHandlerOption
```
WithHandlerLogger provides a logger to be used by the TracingHandler.
If not specified a default logger that discards all output is used.


```go
func WithHandlerRequestBody(bl TraceHandlerRequestBody) TraceHandlerOption
```
WithHandlerRequestBody sets a callback to be invoked to log the request
body. The supplied callback will be called with the request body. The
request body is read and replaced with a new reader, so the next handler in
the chain can still read it.


```go
func WithHandlerRequestBodyJSON(bl TraceHandlerRequestBody) TraceHandlerOption
```


```go
func WithHandlerResponseBody(bl TraceHandlerResponseBody) TraceHandlerOption
```
WithHandlerResponseBody sets a callback to be invoked to log the response
body. The supplied callback will be called with the response body.




### Type TraceHandlerRequestBody
```go
type TraceHandlerRequestBody func(ctx context.Context, logger *slog.Logger, req *http.Request, data []byte)
```
TraceHandlerRequestBody is called to log request body data. The supplied
data is a copy of the original request body.


### Type TraceHandlerResponseBody
```go
type TraceHandlerResponseBody func(ctx context.Context, logger *slog.Logger, req *http.Request, hdr http.Header, statusCode int, data []byte)
```
TraceHandlerResponseBody is called to log response body data. The supplied
data is a copy of the original response body.


### Type TraceHooks
```go
type TraceHooks uint64
```
TraceHooks is a bitmask to control which httptrace hooks are enabled.

### Constants
### TraceGetConn, TraceGotConn, TracePutIdleConn, TraceGotFirstResponseByte, TraceGot100Continue, TraceGot1xxResponse, TraceDNSStart, TraceDNSDone, TraceConnectStart, TraceConnectDone, TraceTLSHandshakeStart, TraceTLSHandshakeDone, TraceWroteHeaderField, TraceWroteHeaders, TraceWait100Continue, TraceWroteRequest, TraceConnections, TraceDNS, TraceConnect, TraceTLS, TraceWrites, TraceResponses, TraceAll
```go
TraceGetConn TraceHooks = 1 << iota
TraceGotConn
TracePutIdleConn
TraceGotFirstResponseByte
TraceGot100Continue
TraceGot1xxResponse
TraceDNSStart
TraceDNSDone
TraceConnectStart
TraceConnectDone
TraceTLSHandshakeStart
TraceTLSHandshakeDone
TraceWroteHeaderField
TraceWroteHeaders
TraceWait100Continue
TraceWroteRequest
// TraceConnections is a convenience group for connection related hooks.
TraceConnections = TraceGetConn | TraceGotConn | TracePutIdleConn
// TraceDNS is a convenience group for DNS hooks.
TraceDNS = TraceDNSStart | TraceDNSDone
// TraceConnect is a convenience group for TCP connection hooks.
TraceConnect = TraceConnectStart | TraceConnectDone
// TraceTLS is a convenience group for TLS handshake hooks.
TraceTLS = TraceTLSHandshakeStart | TraceTLSHandshakeDone
// TraceWrites is a convenience group for request writing hooks.
TraceWrites = TraceWroteHeaderField | TraceWroteHeaders | TraceWait100Continue | TraceWroteRequest
// TraceResponses is a convenience group for response related hooks.
TraceResponses = TraceGotFirstResponseByte | TraceGot100Continue | TraceGot1xxResponse
// TraceAll enables all available trace hooks.
TraceAll TraceHooks = TraceConnections | TraceDNS | TraceConnect | TraceTLS | TraceWrites | TraceResponses

```




### Type TraceRequestBody
```go
type TraceRequestBody func(ctx context.Context, logger *slog.Logger, req *http.Request, data []byte)
```
TraceRequestBody is called to log request body data. The supplied data is a
copy of the original request body.


### Type TraceResponseBody
```go
type TraceResponseBody func(ctx context.Context, logger *slog.Logger, req *http.Request, resp *http.Response, data []byte)
```
TraceResponseBody is called to log response body data. The supplied data is
a copy of the original response body.


### Type TraceRoundtripOption
```go
type TraceRoundtripOption func(*roundtripOptions)
```
TraceRoundtripOption is an option for configuring a TracingRoundTripper.

### Functions

```go
func WithTraceHooks(hooks TraceHooks) TraceRoundtripOption
```
WithTraceHooks sets the trace hooks to be enabled.


```go
func WithTraceRequestBody(bl TraceRequestBody) TraceRoundtripOption
```
WithTraceRequestBody sets a callback to log request body data.


```go
func WithTraceResponseBody(bl TraceResponseBody) TraceRoundtripOption
```
WithTraceResponseBody sets a callback to log response body data.


```go
func WithTracingLogger(logger *slog.Logger) TraceRoundtripOption
```
WithTracingLogger sets the logger to be used for tracing output.




### Type TracingHandler
```go
type TracingHandler struct {
	// contains filtered or unexported fields
}
```
TracingHandler is an http.Handler that wraps another http.Handler to provide
basic request tracing. It logs the start and end of each request and can be
configured to log the request body.

### Functions

```go
func NewTracingHandler(next http.Handler, opts ...TraceHandlerOption) *TracingHandler
```
NewTracingHandler returns a new TracingHandler that wraps the supplied next
http.Handler.



### Methods

```go
func (th *TracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)
```
ServeHTTP implements http.Handler.




### Type TracingRoundTripper
```go
type TracingRoundTripper struct {
	// contains filtered or unexported fields
}
```
TracingRoundTripper is an http.RoundTripper that adds httptrace tracing and
logging capabilities to an underlying RoundTripper.

### Functions

```go
func NewTracingRoundTripper(next http.RoundTripper, opts ...TraceRoundtripOption) *TracingRoundTripper
```
NewTracingRoundTripper creates a new TracingRoundTripper.



### Methods

```go
func (t *TracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error)
```
RoundTrip implements the http.RoundTripper interface.







