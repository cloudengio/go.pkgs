# Package [cloudeng.io/text/testing/testtext](https://pkg.go.dev/cloudeng.io/text/testing/testtext?tab=doc)

```go
import cloudeng.io/text/testing/testtext
```


## Types
### Type Option
```go
type Option func(o *options)
```
Option represents an option to the factory methods in this package.

### Functions

```go
func IncludeControlOpt(v bool) Option
```
IncludeControlOpt controls whether control characters can be included in the
generated strings.




### Type Random
```go
type Random struct {
	// contains filtered or unexported fields
}
```
Random can be used to generate strings containing randomly selected runes.

### Functions

```go
func NewRandom(opts ...Option) *Random
```
NewRandom returns a new instance of Random.



### Methods

```go
func (r Random) AllRuneLens(nRunes int) string
```
AllRuneLens generates a string of length nRunes that contains runes of
differing lengths. The lengths used are a randomized but repeating order of
1..4.


```go
func (r Random) WithRuneLen(nBytes int, nRunes int) string
```
RuneLen generates a string of length nRunes that contains only the requested
number of nBytes (1-4) per rune.







