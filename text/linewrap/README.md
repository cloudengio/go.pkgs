# Package [cloudeng.io/text/linewrap](https://pkg.go.dev/cloudeng.io/text/linewrap?tab=doc)

```go
import cloudeng.io/text/linewrap
```

Package linewrap provides basic support for wrapping text to a given width.

## Functions
### Func Block
```go
func Block(indent, width int, text string) string
```
Block wraps the supplied text to width indented by indent spaces.

### Func Comment
```go
func Comment(indent, width int, comment, text string) string
```
Comment wraps the supplied text to width indented by indent spaces with
each line starting with the supplied comment string. It is intended for
formatting source code comments.

### Func Paragraph
```go
func Paragraph(initial, indent, width int, text string) string
```
Paragraph wraps the supplied text as a 'paragraph' with separate indentation
for the initial and subsequent lines to the specified width.

### Func Prefix
```go
func Prefix(indent int, prefix, text string) string
```
Prefix returns the supplied text with each nonempty line prefixed by indent
spaces and the supplied prefix.

### Func Verbatim
```go
func Verbatim(indent int, text string) string
```
Verbatim returns the supplied text with each nonempty line prefixed by
indent spaces.




