# Package [cloudeng.io/file/diskusage](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc)

```go
import cloudeng.io/file/diskusage
```


## Constants
### Byte, KB, MB, GB, TB, PB, EB, KiB, MiB, GiB, TiB, PiB, EiB
```go
Byte SizeUnit = 1
// base 10
KB = Byte * 1000
MB = KB * 1000
GB = MB * 1000
TB = GB * 1000
PB = TB * 1000
EB = PB * 1000
// base 2 quantities
KiB = Byte << 10
MiB = KiB << 10
GiB = MiB << 10
TiB = GiB << 10
PiB = TiB << 10
EiB = PiB << 10

```



## Functions
### Func BinarySize
```go
func BinarySize(width, precision int, val int64) string
```

### Func DecimalSize
```go
func DecimalSize(width, precision int, val int64) string
```

### Func ParseToBytes
```go
func ParseToBytes(val string) (float64, error)
```



## Types
### Type Binary
```go
type Binary SizeUnit
```
Binary represents a number of bytes in base 2.

### Methods

```go
func (b Binary) Format(f fmt.State, verb rune)
```


```go
func (b Binary) Standardize() (float64, string)
```


```go
func (b Binary) Value(value int64) float64
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




### Type Decimal
```go
type Decimal SizeUnit
```
Decimal represents a number of bytes in base 10.

### Methods

```go
func (b Decimal) Format(f fmt.State, verb rune)
```


```go
func (b Decimal) Standardize() (float64, string)
```


```go
func (b Decimal) Value(value int64) float64
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




### Type SizeUnit
```go
type SizeUnit int64
```
SizeUnit represents a unit of size in bytes. It can be used to represent
both decimal and binary sizes.

### Functions

```go
func BinaryUnitForSize(size int64) SizeUnit
```


```go
func DecimalUnitForSize(size int64) SizeUnit
```



### Methods

```go
func (s SizeUnit) String() string
```


```go
func (s SizeUnit) Value(v int64) float64
```






## Examples
### [ExampleBinary](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc#example-Binary)

### [ExampleDecimal](https://pkg.go.dev/cloudeng.io/file/diskusage?tab=doc#example-Decimal)




