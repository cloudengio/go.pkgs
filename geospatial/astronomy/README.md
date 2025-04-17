# Package [cloudeng.io/geospatial/astronomy](https://pkg.go.dev/cloudeng.io/geospatial/astronomy?tab=doc)

```go
import cloudeng.io/geospatial/astronomy
```


## Functions
### Func ApparentSolarNoon
```go
func ApparentSolarNoon(date datetime.CalendarDate, place datetime.Place) time.Time
```

### Func December
```go
func December(year int) datetime.CalendarDate
```
December returns the winter solstice.

### Func JDEToCalendar
```go
func JDEToCalendar(jde float64) datetime.CalendarDate
```

### Func June
```go
func June(year int) datetime.CalendarDate
```
June returns the summer solstice.

### Func March
```go
func March(year int) datetime.CalendarDate
```
March returns the vernal/spring equinox.

### Func September
```go
func September(year int) datetime.CalendarDate
```
September returns the autumnal equinox.

### Func SunRiseAndSet
```go
func SunRiseAndSet(date datetime.CalendarDate, place datetime.Place) (time.Time, time.Time)
```
SunRiseAndSet returns the time of sunrise and sunset for the specified date,
latitude and longitude.



## Types
### Type Autumn
```go
type Autumn struct{ LocalName string }
```
Autumn implements datetime.DynamicDateRange for the autumn season.

### Methods

```go
func (a Autumn) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (a Autumn) Name() string
```




### Type AutumnEquinox
```go
type AutumnEquinox struct{}
```
AutumnEquinox implements datetime.DynamicDateRange for the autumn equinox.

### Methods

```go
func (s AutumnEquinox) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (s AutumnEquinox) Name() string
```




### Type SolarNoon
```go
type SolarNoon struct{}
```
SolarNoon implements datetime.DynamicTimeOfDay for the solar noon (aka
Zenith).

### Methods

```go
func (s SolarNoon) Evaluate(cd datetime.CalendarDate, place datetime.Place) datetime.TimeOfDay
```


```go
func (s SolarNoon) Name() string
```




### Type Spring
```go
type Spring struct{}
```
Spring implements datetime.DynamicDateRange for the spring season.

### Methods

```go
func (s Spring) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (s Spring) Name() string
```




### Type SpringEquinox
```go
type SpringEquinox struct{}
```
SpringEquinox implements datetime.DynamicDateRange for the spring equinox.

### Methods

```go
func (s SpringEquinox) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (s SpringEquinox) Name() string
```




### Type Summer
```go
type Summer struct{}
```
Summer implements datetime.DynamicDateRange for the summer season.

### Methods

```go
func (s Summer) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (s Summer) Name() string
```




### Type SummerSolstice
```go
type SummerSolstice struct{}
```
SummerSolstice implements datetime.DynamicDateRange for the summer solstice.

### Methods

```go
func (s SummerSolstice) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (s SummerSolstice) Name() string
```




### Type SunRise
```go
type SunRise struct{}
```
SunRise implements datetime.DynamicTimeOfDay for sunrise.

### Methods

```go
func (s SunRise) Evaluate(cd datetime.CalendarDate, place datetime.Place) datetime.TimeOfDay
```


```go
func (s SunRise) Name() string
```




### Type SunSet
```go
type SunSet struct{}
```
SunSet implements datetime.DynamicTimeOfDay for sunset.

### Methods

```go
func (s SunSet) Evaluate(cd datetime.CalendarDate, place datetime.Place) datetime.TimeOfDay
```


```go
func (s SunSet) Name() string
```




### Type Winter
```go
type Winter struct{}
```
Winter implements datetime.DynamicDateRange for the winter season.

### Methods

```go
func (w Winter) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (w Winter) Name() string
```




### Type WinterSolstice
```go
type WinterSolstice struct{}
```
WinterSolstice implements datetime.DynamicDateRange for the winter solstice.

### Methods

```go
func (s WinterSolstice) Evaluate(year int) datetime.CalendarDateRange
```


```go
func (s WinterSolstice) Name() string
```







