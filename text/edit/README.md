# Package [cloudeng.io/text/edit](https://pkg.go.dev/cloudeng.io/text/edit?tab=doc)

```go
import cloudeng.io/text/edit
```

Package edit provides support for editing in-memory byte slices using
insert, delete and replace operations.

## Functions
### Func Do
```go
func Do(contents []byte, deltas ...Delta) []byte
```
Do applies the supplied deltas to contents as follows:
 1. Deltas are sorted by their start position, then at each position,
 2. deletions are applied, then
 3. replacements are applied, then,
 4. insertions are applied.

Sorting is stable with respect the order specified in the function
invocation. Multiple deletions and replacements overwrite each other,
whereas insertions are concatenated. All position values are with respect to
the original value of contents.

### Func DoString
```go
func DoString(contents string, deltas ...Delta) string
```
DoString is like Do but for strings.

### Func Validate
```go
func Validate(contents []byte, deltas ...Delta) error
```
Validate determines if the supplied deltas fall within the bounds of
content.



## Types
### Type Delta
```go
type Delta struct {
	// contains filtered or unexported fields
}
```
Delta represents an insertion, deletion or replacement.

### Functions

```go
func Delete(pos, size int) Delta
```
Delete creates a Delta to delete size bytes starting at pos.


```go
func Insert(pos int, data []byte) Delta
```
Insert creates a Delta to insert the supplied bytes at pos.


```go
func InsertString(pos int, text string) Delta
```
InsertString is like Insert but for a string.


```go
func Replace(pos, size int, data []byte) Delta
```
Replace creates a Delta to replace size bytes starting at pos with text.
The string may be shorter or longer than size.


```go
func ReplaceString(pos, size int, text string) Delta
```
ReplaceString is like Replace but for a string.



### Methods

```go
func (d Delta) String() string
```
String implements stringer. The format is as follows:

    deletions:    < (from, to]
    insertions:   > @pos#<num bytes>
    replacements: ~ @pos#<num-bytes>/<num-bytes>


```go
func (d Delta) Text() string
```
Text returns the text associated with the Delta.






## Examples
### [ExampleDo](https://pkg.go.dev/cloudeng.io/text/edit?tab=doc#example-Do)




