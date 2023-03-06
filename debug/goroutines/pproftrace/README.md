# Package [cloudeng.io/debug/goroutines/pproftrace](https://pkg.go.dev/cloudeng.io/debug/goroutines/pproftrace?tab=doc)

```go
import cloudeng.io/debug/goroutines/pproftrace
```


## Functions
### Func Format
```go
func Format(key, value string) (string, error)
```
Format returns a nicely formatted dump of the goroutines with the pprof
key/value label.

### Func LabelExists
```go
func LabelExists(key, value string) (bool, error)
```
LabelExists returns true if a goroutine with the pprof key/value label
exists.

### Func Run
```go
func Run(ctx context.Context, key, value string, runner func(context.Context))
```
Run uses pprof's label support to attach the specified key/value label
to all goroutines spawed by the supplied runner. Run returns when runner
returns.




