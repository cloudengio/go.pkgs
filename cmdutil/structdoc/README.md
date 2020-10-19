# Package [cloudeng.io/cmdutil/structdoc](https://pkg.go.dev/cloudeng.io/cmdutil/structdoc?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/cmdutil/structdoc)](https://goreportcard.com/report/cloudeng.io/cmdutil/structdoc)

```go
import cloudeng.io/cmdutil/structdoc
```

Package structdoc provides a means of exposing struct tags for use when
generating documentation for those structs.

## Functions
### Func FormatFields
```go
func FormatFields(prefix, indent int, fields []Field) string
```
FormatFields formats the supplied fields as follows:

    <prefix><name>:<padding><text>

where padding is calculated so as to line up the text. Prefix sets the
number of spaces to be prefixed and indent increases the prefix for each sub
field.

### Func TypeName
```go
func TypeName(t interface{}) string
```
TypeName returns the fully qualified name of the supplied type or the string
representation of an anonymous type.



## Types
### Type Description
```go
type Description struct {
	Detail string
	Fields []Field
}
```
Description represents a structured description of a struct type based on
struct tags. The Detail field may be supplied when constructing the
description.

### Functions

```go
func Describe(t interface{}, tag, detail string) (*Description, error)
```
Describe generates a Description for the supplied type based on its struct
tags. Detail can be used to provide a top level of detail, such as the type
name and a summary.



### Methods

```go
func (d *Description) String() string
```
String returns a string representation of the description.




### Type Field
```go
type Field struct {
	// Name is the name of the original field. The name takes
	// into account any name specified via a json or yaml tag.
	Name string
	// Doc is the text extracted from the struct tag for this field.
	Doc string
	// Slice is true if this field is a slice.
	Slice bool
	// Fields, if this field is a struct, contains descriptions for
	// any documented fields in that struct.
	Fields []Field `json:",omitempty" yaml:",omitempty"`
}
```
Field represents the description of a field and any similarly tagged
subfields.





