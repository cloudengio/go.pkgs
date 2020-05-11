# Package [cloudeng.io/text/linewrap](https://pkg.go.dev/cloudeng.io/text/linewrap?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/text/linewrap)](https://goreportcard.com/report/cloudeng.io/text/linewrap)

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
Comment wraps the supplied text to width indented by indent spaces with each
line starting with the supplied comment string. It is intended for
formatting source code comments.

### Func Paragraph
```go
func Paragraph(initial, indent, width int, text string) string
```
Paragraph wraps the supplied text as a 'paragraph' with separate indentation
for the initial and subsequent lines to the specified width.




