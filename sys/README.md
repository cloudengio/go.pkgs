# Package [cloudeng.io/sys](https://pkg.go.dev/cloudeng.io/sys?tab=doc)

```go
import cloudeng.io/sys
```

Package sys provides system-level utilities that are supported across
different operating systems.

## Functions
### Func AvailableBytes
```go
func AvailableBytes(filename string) (int64, error)
```
AvailableBytes returns the number of available bytes on the filesystem where
the file is located.




