# Package [cloudeng.io/file/filewalk/localfs](https://pkg.go.dev/cloudeng.io/file/filewalk/localfs?tab=doc)

```go
import cloudeng.io/file/filewalk/localfs
```


## Functions
### Func New
```go
func New() filewalk.FS
```

### Func NewLevelScanner
```go
func NewLevelScanner(path string) filewalk.LevelScanner
```



## Types
### Type T
```go
type T struct{ file.FS }
```
T represents an instance of filewalk.FS for a local filesystem.

### Methods

```go
func (l *T) LevelScanner(prefix string) filewalk.LevelScanner
```







