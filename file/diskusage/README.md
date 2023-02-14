# Package [cloudeng.io/file/diskusage](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc)

```go
import cloudeng.io/file/diskusage
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




### Type Calculator
```go
type Calculator interface {
	Calculate(int64) int64
	String() string
}
```

### Functions

```go
func NewIdentity() Calculator
```


```go
func NewRAID0(stripeSize int64, numStripes int) Calculator
```


```go
func NewSimple(blocksize int64) Calculator
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

### Methods

```go
func (i Identity) Calculate(size int64) int64
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

### Methods

```go
func (r0 RAID0) Calculate(size int64) int64
```


```go
func (r0 RAID0) String() string
```




### Type Simple
```go
type Simple struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (s Simple) Calculate(size int64) int64
```


```go
func (s Simple) String() string
```






## Examples
### [ExampleBase2Bytes](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc#example-Base2Bytes)

### [ExampleDecimalBytes](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc#example-DecimalBytes)




