# Package [cloudeng.io/file/diskusage](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc)

```go
import cloudeng.io/file/diskusage
```


## Functions
### Func ParseToBytes
```go
func ParseToBytes(val string) (float64, error)
```



## Types
### Type Base2Bytes
```go
type Base2Bytes int64
```
Base2Bytes represents a number of bytes in base 2.

### Constants
### KiB, MiB, GiB, TiB, PiB, EiB
```go
KiB Base2Bytes = 1024
MiB Base2Bytes = KiB * 1024
GiB Base2Bytes = MiB * 1024
TiB Base2Bytes = GiB * 1024
PiB Base2Bytes = TiB * 1024
EiB Base2Bytes = PiB * 1024

```
Values for Base2Bytes.



### Methods

```go
func (b Base2Bytes) Num(value int64) float64
```


```go
func (b Base2Bytes) Standardize() (float64, string)
```




### Type Block
```go
type Block struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (s Block) Calculate(_, blocks int64) int64
```


```go
func (s Block) String() string
```




### Type Calculator
```go
type Calculator interface {
	Calculate(bytes, blocks int64) int64
	String() string
}
```
Calculator is used to calculate the size of a file or directory based on
either its size in bytes (often referred to as its apparent size) and/or
the number of storage blocks it occupies. Some file systems support sparse
files (most unix filesystems) where the number of blocks occupied by a file
is less than the number of bytes it represents, hence, the term 'apparent
size'.

### Functions

```go
func NewBlock(blocksize int64) Calculator
```
Block uses the number of blocks occupied by a file to calculate its size.


```go
func NewIdentity() Calculator
```


```go
func NewRAID0(stripeSize int64, numStripes int) Calculator
```


```go
func NewRoundup(blocksize int64) Calculator
```




### Type DecimalBytes
```go
type DecimalBytes int64
```
Base2Bytes represents a number of bytes in base 10.

### Constants
### KB, MB, GB, TB, PB, EB
```go
KB DecimalBytes = 1000
MB DecimalBytes = KB * 1000
GB DecimalBytes = MB * 1000
TB DecimalBytes = GB * 1000
PB DecimalBytes = TB * 1000
EB DecimalBytes = PB * 1000

```
Values for DecimalBytes.



### Methods

```go
func (b DecimalBytes) Num(value int64) float64
```


```go
func (b DecimalBytes) Standardize() (float64, string)
```




### Type Identity
```go
type Identity struct{}
```
Identity returns the apparent size of a file.

### Methods

```go
func (i Identity) Calculate(bytes, _ int64) int64
```


```go
func (i Identity) String() string
```




### Type RAID0
```go
type RAID0 struct {
	// contains filtered or unexported fields
}
```
RAID0 is a calculator for RAID0 volumes based on the apparent size of the
file and the RAID0 stripe size and number of stripes.

### Methods

```go
func (r0 RAID0) Calculate(size, _ int64) int64
```


```go
func (r0 RAID0) String() string
```




### Type Roundup
```go
type Roundup struct {
	// contains filtered or unexported fields
}
```
Roundup rounds up the apparent size of a file to the nearest block size
multiple.

### Methods

```go
func (s Roundup) Calculate(bytes, _ int64) int64
```


```go
func (s Roundup) String() string
```






## Examples
### [ExampleBase2Bytes](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc#example-Base2Bytes)

### [ExampleDecimalBytes](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc#example-DecimalBytes)




