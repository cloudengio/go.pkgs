# Package [cloudeng.io/cmdutil/cmdjson](https://pkg.go.dev/cloudeng.io/cmdutil/cmdjson?tab=doc)

```go
import cloudeng.io/cmdutil/cmdjson
```


## Types
### Type FlexTime
```go
type FlexTime time.Time
```
FlexTime is a time.Time that can be unmarshaled from time.RFC3339,
time.DateTime, time.TimeOnly or time.DateOnly formats. It is always
marshaled to time.RFC3339.

### Methods

```go
func (t FlexTime) MarshalJSON() ([]byte, error)
```


```go
func (t FlexTime) String() string
```


```go
func (t *FlexTime) UnmarshalJSON(data []byte) error
```




### Type Permissions
```go
type Permissions = cmdtypes.Permissions
```
Permissions is an alias for cmdtypes.Permissions, which is decoded from JSON
via its UnmarshalText and UnmarshalJSON methods. It is retained here for the
convenience of code that already refers to it by this name.

### Functions

```go
func ParsePermissions(s string) (Permissions, error)
```
ParsePermissions parses a permission string in octal, rwx, or symbolic
format, as per cmdtypes.ParsePermissions.




### Type RFC3339Time
```go
type RFC3339Time time.Time
```
RFC3339Time is a time.Time that marshals to and from RFC3339 format.

### Methods

```go
func (t RFC3339Time) MarshalJSON() ([]byte, error)
```


```go
func (t RFC3339Time) String() string
```


```go
func (t *RFC3339Time) UnmarshalJSON(data []byte) error
```







