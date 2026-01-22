# Package [cloudeng.io/text/textutil](https://pkg.go.dev/cloudeng.io/text/textutil?tab=doc)

```go
import cloudeng.io/text/textutil
```

Package textutil provides utility routines for working with text,
in particular utf8 encoded text.

## Functions
### Func BytesToString
```go
func BytesToString(b []byte) string
```
BytesToString returns a string with the supplied byte slice as its
contents. The original byte slice must never be modified. Taken from
strings.Builder.String().

### Func ReverseBytes
```go
func ReverseBytes(input string) []byte
```
ReverseBytes returns a new slice containing the runes in the input string in
reverse order.

### Func ReverseString
```go
func ReverseString(input string) string
```
ReverseString is like ReverseBytes but returns a string.

### Func StringToBytes
```go
func StringToBytes(s string) []byte
```
StringToBytes returns the byte slice containing the data for the
supplied string without any allocations or copies. It should only
be used when the resulting byte slice will never be modified. See
https://groups.google.com/g/golang-nuts/c/Zsfk-VMd_fU/m/O1ru4fO-BgAJ

### Func TrimUnicodeQuotes
```go
func TrimUnicodeQuotes(text string) string
```
TrimUnicodeQuotes trims leading and trailing UTF-8 curly quotes from text
using unicode properties (Pi and Pf).



## Types
### Type RewriteRule
```go
type RewriteRule struct {
	Match       *regexp.Regexp
	Replacement string
}
```
RewriteRule represents a rewrite rule of the form s/<match>/<replace>/ or
s%<match>%<replace>%. Separators can be escpaed using a \.

### Functions

```go
func NewRewriteRule(rule string) (RewriteRule, error)
```
NewReplacement accepts a string of the form s/<match-re>/<replacement>/ or
s%<match-re>%<replacement>% and returns a RewriteRule that can be used to
perform the rewquested rewrite. Separators can be escpaed using a \.



### Methods

```go
func (rr RewriteRule) MatchString(input string) bool
```
Match applies regexp.MatchString.


```go
func (rr RewriteRule) ReplaceAllString(input string) string
```
ReplaceAllString(input string) applies regexp.ReplaceAllString.




### Type RewriteRules
```go
type RewriteRules []RewriteRule
```

### Functions

```go
func NewRewriteRules(rules ...string) (RewriteRules, error)
```



### Methods

```go
func (rw RewriteRules) ReplaceAllStringFirst(input string) string
```







