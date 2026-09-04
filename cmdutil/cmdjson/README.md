# Package [cloudeng.io/cmdutil/cmdjson](https://pkg.go.dev/cloudeng.io/cmdutil/cmdjson?tab=doc)

```go
import cloudeng.io/cmdutil/cmdjson
```


## Types
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
func (t RFC3339Time) MarshalJSONTo(enc *jsontext.Encoder) error
```
MarshalJSONTo implements json.MarshalerTo from encoding/json/v2, writing the
time directly to the encoder rather than through an intermediate value.


```go
func (t RFC3339Time) String() string
```


```go
func (t *RFC3339Time) UnmarshalJSON(data []byte) error
```


```go
func (t *RFC3339Time) UnmarshalJSONFrom(dec *jsontext.Decoder) error
```
UnmarshalJSONFrom implements json.UnmarshalerFrom from encoding/json/v2.
A null leaves the value unchanged, as it does for UnmarshalJSON.







