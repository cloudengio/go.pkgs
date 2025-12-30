# Package [cloudeng.io/logging/ctxlog](https://pkg.go.dev/cloudeng.io/logging/ctxlog?tab=doc)

```go
import cloudeng.io/logging/ctxlog
```

Package ctxlog provides a context key and functions for logging to a
context.

## Functions
### Func Debug
```go
func Debug(ctx context.Context, msg string, args ...any)
```

### Func Error
```go
func Error(ctx context.Context, msg string, args ...any)
```

### Func Info
```go
func Info(ctx context.Context, msg string, args ...any)
```

### Func Log
```go
func Log(ctx context.Context, level slog.Level, msg string, args ...any)
```

### Func LogDepth
```go
func LogDepth(ctx context.Context, logger *slog.Logger, level slog.Level, depth int, msg string, args ...any)
```
LogDepth logs a message at the specified level with the caller information
adjusted by the provided depth.

### Func Logger
```go
func Logger(ctx context.Context) *slog.Logger
```
Logger returns the logger from the given context. If no logger is set,
it returns a discard logger.

### Func NewJSONLogger
```go
func NewJSONLogger(ctx context.Context, w io.Writer, opts *slog.HandlerOptions) context.Context
```
NewJSONLogger returns a new context with a JSON logger.

### Func NewLogLogger
```go
func NewLogLogger(ctx context.Context, level slog.Level) *log.Logger
```
NewLogLogger returns a new standard library logger that logs to the provided
context's logger at the specified level.

### Func Warn
```go
func Warn(ctx context.Context, msg string, args ...any)
```

### Func WithAttributes
```go
func WithAttributes(ctx context.Context, attributes ...any) context.Context
```
WithAttributes returns a new context with the embedded logger updated with
the given logger attributes.

### Func WithLogger
```go
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context
```
WithLogger returns a new context with the given logger.



## Examples
### [ExampleLogger](https://pkg.go.dev/cloudeng.io/logging/ctxlog?tab=doc#example-Logger)

### [ExampleNewJSONLogger](https://pkg.go.dev/cloudeng.io/logging/ctxlog?tab=doc#example-NewJSONLogger)




