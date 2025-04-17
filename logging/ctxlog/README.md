# Package [cloudeng.io/logging/ctxlog](https://pkg.go.dev/cloudeng.io/logging/ctxlog?tab=doc)

```go
import cloudeng.io/logging/ctxlog
```

Package ctxlog provides a context key and functions for logging to a
context.

## Functions
### Func Context
```go
func Context(ctx context.Context, logger *slog.Logger) context.Context
```
ContextWithLogger returns a new context with the given logger.

### Func ContextWith
```go
func ContextWith(ctx context.Context, attributes ...any) context.Context
```
ContextWithLoggerAttributes returns a new context with the embedded logger
updated with the given logger attributes.

### Func Logger
```go
func Logger(ctx context.Context) *slog.Logger
```
LoggerFromContext returns the logger from the given context. If no logger is
set, it returns a discard logger.

### Func NewJSONLogger
```go
func NewJSONLogger(ctx context.Context, w io.Writer, opts *slog.HandlerOptions) context.Context
```
NewJSONLogger returns a new context with a JSON logger.



## Examples
### [ExampleLogger](https://pkg.go.dev/cloudeng.io/logging/ctxlog?tab=doc#example-Logger)

### [ExampleNewJSONLogger](https://pkg.go.dev/cloudeng.io/logging/ctxlog?tab=doc#example-NewJSONLogger)




