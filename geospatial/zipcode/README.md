# Package [cloudeng.io/geospatial/zipcode](https://pkg.go.dev/cloudeng.io/geospatial/zipcode?tab=doc)

```go
import cloudeng.io/geospatial/zipcode
```

Zipcode lookups using data from www.geonames.org.

## Types
### Type DB
```go
type DB struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewDB() *DB
```



### Methods

```go
func (zdb *DB) LatLong(admin, postal string) (LatLong, bool)
```
LatLong returns the latitude and longitude for the specified postal code
and admin code (eg. AK 99553). GB and CA postal codes come in two formats,
either the short form or long form:

    GB: Eng BN91, or Eng "BN91 9AA".
    CA: AB T0A, or AB "T0A 0A0".


```go
func (zdb *DB) Load(data []byte, _ ...Option) error
```




### Type LatLong
```go
type LatLong struct {
	Lat  float64 // Estimated latitude (wgs84)
	Long float64 // Estimated longitude (wgs84)
}
```


### Type Option
```go
type Option func(o *options)
```

### Functions

```go
func WithTag(tag string) Option
```







