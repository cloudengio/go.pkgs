# Package [cloudeng.io/datetime](https://pkg.go.dev/cloudeng.io/datetime?tab=doc)

```go
import cloudeng.io/datetime
```

Package datetime provides support for working with dates, the time of day
and associated ranges.

## Variables
### ErrInvalidISO8601Duration
```go
ErrInvalidISO8601Duration = errors.New("invalid ISO8601 duration")

```



## Functions
### Func AsISO8601Period
```go
func AsISO8601Period(dur time.Duration) string
```

### Func ContextWithYearPlace
```go
func ContextWithYearPlace(ctx context.Context, yp YearPlace) context.Context
```
ContextWithYearPlace returns a new context with the given YearPlace value
stored in it.

### Func DaysInFeb
```go
func DaysInFeb(year int) uint8
```
DaysInFeb returns the number of days in February for the given year.

### Func DaysInMonth
```go
func DaysInMonth(year int, month Month) uint8
```
DaysInMonth returns the number of days in the given month for the given
year.

### Func DaysInYear
```go
func DaysInYear(year int) int
```
DaysInYear returns the number of days in the given year.dc

### Func IsLeap
```go
func IsLeap(year int) bool
```
IsLeap returns true if the given year is a leap year.

### Func ParseISO8601Period
```go
func ParseISO8601Period(dur string) (time.Duration, error)
```
ParseISO8601Period parses an ISO 8601 'period' of the form [-]PnYnMnDTnHnMnS

### Func Time
```go
func Time(yp YearPlace, date Date, tod TimeOfDay) time.Time
```
Time returns a time.Time for the specified YearAndPlace, Date and TimeOfDay
using time.Date, and hence, implicitly normalizing the date.



## Types
### Type CalendarDate
```go
type CalendarDate uint32
```
CalendarDate represents a date with a year, month and day. Year is
represented in the top 16 bits and Date in the lower 16 bits.

### Functions

```go
func CalendarDateFromTime(t time.Time) CalendarDate
```
CalendarDateFromTime creates a new CalendarDate from the specified
time.Time.


```go
func NewCalendarDate(year int, month Month, day int) CalendarDate
```
NewCalendarDate creates a new CalendarDate from the specified year, month
and day. Year must be in the range 0..65535, month in the range 0..12 and
day in the range 0..31.


```go
func NewCalendarDateFromTime(t time.Time) CalendarDate
```
NewCalendarDateFromTime creates a new CalendarDate from the specified
time.Time.


```go
func ParseAnyDate(year int, val string) (CalendarDate, error)
```
ParseAnyDate parses a date in the format '01/02/2006', 'Jan-02-2006',
'01/02', 'Jan-02' or '01'. The year argument is ignored for the '01/02/2006'
and 'Jan-02-2006' formats. Jan-02, 01/02 are treated as month and day and
the year argument is used to set the year.


```go
func ParseCalendarDate(val string) (CalendarDate, error)
```
ParseCalendarDate a numeric calendar date in formats 'Jan-02-2006' with
error checking for valid month and day.


```go
func ParseNumericCalendarDate(val string) (CalendarDate, error)
```
ParseNumericCalendarDate a numeric calendar date in formats '01/02/2006'
with error checking for valid month and day.



### Methods

```go
func (cd CalendarDate) Date() Date
```
Date returns the Date for the CalendarDate.


```go
func (cd CalendarDate) Day() int
```


```go
func (cd CalendarDate) DayOfYear() int
```


```go
func (cd CalendarDate) IsDST(loc *time.Location) bool
```
IsDST returns true if the date is within daylight savings time for the
specified location assuming that the time is 12:00 hours. DST starts at 2am
and ends at 3am.


```go
func (cd CalendarDate) Month() Month
```


```go
func (cd CalendarDate) Normalize(firstOfMonth bool) CalendarDate
```
Normalize adjusts the date for the given year. If the day is zero
then firstOfMonth is used to determine the interpretation of the day.
If firstOfMonth is true then the day is set to the first day of the month,
otherwise it is set to the last day of the month. Month is normalized to be
in the range 1-12.


```go
func (cd *CalendarDate) Parse(val string) error
```


```go
func (cd CalendarDate) String() string
```


```go
func (cd CalendarDate) Time(tod TimeOfDay, loc *time.Location) time.Time
```


```go
func (cd CalendarDate) Tomorrow() CalendarDate
```
Tomorrow returns the CalendarDate for the day after the specified date,
wrapping to the next month or year as needed.


```go
func (cd CalendarDate) Year() int
```


```go
func (cd CalendarDate) YearDay() YearDay
```


```go
func (cd CalendarDate) Yesterday() CalendarDate
```
Yesterday returns the CalendarDate for the day before the specified date,
wrapping to the previous month or year as needed.




### Type CalendarDateList
```go
type CalendarDateList []CalendarDate
```
CalendarDateList represents a list of CalendarDate values, it can sorted and
searched using the slices package.

### Methods

```go
func (cdl CalendarDateList) Merge() CalendarDateRangeList
```
Merge returns a new list of date ranges that contains merged
consecutive calendar dates into ranges. The dates are normalized using
date.Normalize(true). The date list is assumed to be sorted.


```go
func (cdl CalendarDateList) String() string
```




### Type CalendarDateRange
```go
type CalendarDateRange uint64
```
CalendarDateRange represents a range of CalendarDate values.

### Functions

```go
func NewCalendarDateRange(from, to CalendarDate) CalendarDateRange
```
NewCalendarDateRange returns a CalendarDateRange for the from/to dates. If
the from date is later than the to date then they are swapped. The resulting
from and to dates are then normalized using calendardate.Normalize(year,
true) for the from date and calendardate.Normalize(year, false) for the to
date.



### Methods

```go
func (cdr CalendarDateRange) Bound(bound CalendarDateRange) CalendarDateRange
```
Bound returns a new CalendarDateRange that is bounded by the specified
CalendarDateRange, namely the from date is the later of the two from dates
and the to date is the earlier of the two to dates. If the resulting range
is empty then the zero value is returned.


```go
func (cdr CalendarDateRange) DateRange(year int) DateRange
```


```go
func (cdr CalendarDateRange) Dates() iter.Seq[CalendarDate]
```
Dates returns an iterator that yields each Date in the range for the given
year.


```go
func (cdr CalendarDateRange) DatesConstrained(dc Constraints) iter.Seq[CalendarDate]
```
DatesConstrained returns an iterator that yields each Date in the range for
the given year constrained by the given DateConstraints.


```go
func (cdr CalendarDateRange) Days() iter.Seq[YearDay]
```
Days returns an iterator that yields each day in the range for the given
year.


```go
func (cdr CalendarDateRange) DaysConstrained(dc Constraints) iter.Seq[YearDay]
```
DaysConstrained returns an iterator that yields each day in the range for
the given year constrained by the given DateConstraints.


```go
func (cdr CalendarDateRange) From() CalendarDate
```
From returns the start date of the range for the specified year. Feb 29 is
returned as Feb 28 for non-leap years.


```go
func (cd CalendarDateRange) Include(d CalendarDate) bool
```
Include returns true if the specified date is within the range.


```go
func (cdr *CalendarDateRange) Parse(val string) error
```
Parse ranges in formats '01/2006:03/2007', 'Jan-2006:Mar-2007',
'01/02/2006:03/04/2007' or 'Jan-02-2006:Mar-04-2007', If the from day is
zero then it is treated as the first day of the month. If the from day is 29
for a non-leap year then it is left as 29. If the to day is zero then it is
treated as the last day of the month taking the year into account for Feb.
The start date must be before the end date after normalization as per the
above rules.


```go
func (cdr CalendarDateRange) RangesConstrained(dc Constraints) iter.Seq[CalendarDateRange]
```
Ranges returns an iterator that yields each DateRange in the range for the
given year constrained by the given DateConstraints.


```go
func (cdr CalendarDateRange) String() string
```


```go
func (cdr CalendarDateRange) To() CalendarDate
```
To returns the end date of the range for the specified year. Feb 29 is
returned as Feb 28 for non-leap years.


```go
func (cdr CalendarDateRange) Truncate(year int) DateRange
```
Truncate returns a DateRange that is truncated to the start or end of
specified year iff the range spans consecutive years, otherwise it returns
DateRange(0).




### Type CalendarDateRangeList
```go
type CalendarDateRangeList []CalendarDateRange
```

### Methods

```go
func (cdrl CalendarDateRangeList) Bound(bound CalendarDateRange) CalendarDateRangeList
```
Bound returns a new list of date ranges that are bounded by the supplied
calendar date range.


```go
func (cdrl CalendarDateRangeList) Merge() CalendarDateRangeList
```
Merge returns a new list of date ranges that contains merged consecutive
overlapping ranges. The date list is assumed to be sorted.


```go
func (cdrl CalendarDateRangeList) MergeMonths(year int, months MonthList) CalendarDateRangeList
```
MergeMonths returns a merged list of date ranges that contains the specified
months for the given year.


```go
func (cdrl *CalendarDateRangeList) Parse(ranges []string) error
```
Parse parses a list of ranges in the format expected by
CalendarDateRange.Parse.




### Type Constraints
```go
type Constraints struct {
	Weekdays       bool                 // If true, include weekdays
	Weekends       bool                 // If true, include weekends
	Custom         DateList             // If non-empty, exclude these dates
	CustomCalendar CalendarDateList     // If non-empty, exclude these calendar dates
	Dynamic        DynamicDateRangeList // If non-nil, exclude dates based on the evaluation of the dynamic date range functions.
}
```
Constraints represents constraints on date values such as weekends or custom
dates to exclude. Custom dates take precedence over weekdays and weekends.

### Methods

```go
func (dc Constraints) Empty() bool
```


```go
func (dc Constraints) Include(when time.Time) bool
```
Include returns true if the given date satisfies the constraints. Custom
dates are evaluated before weekdays and weekends. An empty set Constraints
will return true, ie. include all dates.


```go
func (dc Constraints) String() string
```




### Type Date
```go
type Date uint16
```
Date as a uint16 with the month in the high byte and the day in the low
byte.

### Functions

```go
func DateFromTime(when time.Time) Date
```
DateFromTime returns the Date for the given time.Time.


```go
func NewDate(month Month, day int) Date
```
NewDate returns a Date for the given month and day. Both are assumed to
valid for the context in which they are used. Normalize should be used to to
adjust for a given year and interpretation of zero day value.


```go
func ParseDate(val string) (Date, error)
```
ParseDate parses a date in the forma 'Jan-02' with error checking for valid
month and day (Feb is treated as having 29 days)


```go
func ParseNumericDate(val string) (Date, error)
```
Parse a numeric date in the format '01/02' with error checking for valid
month and day (Feb is treated as having 29 days)



### Methods

```go
func (d Date) CalendarDate(year int) CalendarDate
```


```go
func (d Date) Day() int
```
Day returns the day for the date.


```go
func (d Date) DayOfYear(year int) int
```
DayOfYear returns the day of the year for the given year as 1-365 for
non-leap years and 1-366 for leap years. It will silently treat days that
exceed those for a given month to the last day of that month. A day of zero
can be used to refer to the last day of the previous month.


```go
func (d Date) Month() Month
```
Month returns the month for the date.


```go
func (d Date) Normalize(year int, firstOfMonth bool) Date
```
Normalize adjusts the date for the given year. If the day is zero
then firstOfMonth is used to determine the interpretation of the day.
If firstOfMonth is true then the day is set to the first day of the month,
otherwise it is set to the last day of the month. Month is normalized to be
in the range 1-12.


```go
func (d *Date) Parse(val string) error
```
Parse date in formats '01', 'Jan','01/02' or 'Jan-02' with error checking
for valid month and day (Feb is treated as having 29 days)


```go
func (d Date) String() string
```


```go
func (d Date) Tomorrow(year int) Date
```
Tomorrow returns the date of the next day. It will silently treat days that
exceed those for a given month as the last day of that month. 12/31 wraps to
1/1.


```go
func (d Date) YearDay(year int) YearDay
```


```go
func (d Date) Yesterday(year int) Date
```
Yesterday returns the date of the previous day. 1/1 wraps to 12/31.




### Type DateList
```go
type DateList []Date
```
DateList represents a list of Dates, it can be sorted and searched the
slices package.

### Methods

```go
func (dl DateList) ExpandMonths(year int) DateRangeList
```


```go
func (dl DateList) Merge(year int) DateRangeList
```
Merge returns a new list of date ranges that contains merged consecutive
dates into ranges. All dates are normalized using date.Normalize(year,
true). The date list is assumed to be sorted.


```go
func (dl *DateList) Parse(val string) error
```
Parse a comma separated list of Dates.


```go
func (dl DateList) String() string
```




### Type DateRange
```go
type DateRange uint32
```
DateRange represents a range of dates, inclusive of the start and end dates.
NewDateRange and Parse create or initialize a DateRange. The from and to
months are stored in the top 8 bits and the from and to days in the lower 8
bits to allow for sorting.

### Functions

```go
func DateRangeYear() DateRange
```
DateRangeYear returns a DateRange for the entire year.


```go
func NewDateRange(from, to Date) DateRange
```
NewDateRange returns a DateRange for the from/to dates for the specified
year. If the from date is later than the to date then they are swapped.



### Methods

```go
func (dr DateRange) Bound(year int, bound DateRange) DateRange
```
Bound returns a new DateRange that is bounded by the specified DateRange,
namely the from date is the later of the two from dates and the to date is
the earlier of the two to dates. If the resulting range is empty then the
zero value is returned.


```go
func (dr DateRange) CalendarDateRange(year int) CalendarDateRange
```
CalendarDateRange returns a CalendarDateRange for the DateRange for the
specified year. The date range is first normalized to the specified year
before creating the CalendarDateRange.


```go
func (dr DateRange) Dates(year int) func(yield func(CalendarDate) bool)
```
Dates returns an iterator that yields each CalendarDate in the range for the
given year. All of the CalendarDate values will have the same year.


```go
func (dr DateRange) DatesConstrained(year int, dc Constraints) func(yield func(CalendarDate) bool)
```
DatesConstrained returns an iterator that yields each CalendarDate in the
range for the given year constrained by the given DateConstraints. All of
the CalendarDate values will have the same year.


```go
func (dr DateRange) Days(year int) func(yield func(YearDay) bool)
```
Days returns an iterator that yields each day in the range for the given
year.


```go
func (dr DateRange) DaysConstrained(year int, dc Constraints) func(yield func(YearDay) bool)
```
DaysConstrained returns an iterator that yields each day in the range for
the given year constrained by the given DateConstraints.


```go
func (dr DateRange) Equal(year int, dr2 DateRange) bool
```
Equal returns true if the two DateRange values are equal for the given year.
Both ranges are first normalized before comparison.


```go
func (dr DateRange) From(year int) Date
```
From returns the start date of the range for the specified year. Feb 29 is
returned as Feb 28 for non-leap years.


```go
func (dr DateRange) Include(d Date) bool
```
Include returns true if the specified date is within the range.


```go
func (dr DateRange) Normalize(year int) DateRange
```
Normalize rerturns a new DateRange with the from and to dates normalized
to the specified year. This is equivalent to calling date.Normalize(year,
true) for the from date and date.Normalize(year, false) for the to date.


```go
func (dr *DateRange) Parse(val string) error
```
Parse ranges in formats '01:03', 'Jan:Mar', '01/02:03-04' or
'Jan-02:Mar-04'.


```go
func (dr DateRange) RangesConstrained(year int, dc Constraints) func(yield func(CalendarDateRange) bool)
```
Ranges returns an iterator that yields each DateRange in the range for the
given year constrained by the given DateConstraints.


```go
func (dr DateRange) String() string
```


```go
func (dr DateRange) To(year int) Date
```
To returns the end date of the range for the specified year. Feb 29 is
returned as Feb 28 for non-leap years.




### Type DateRangeList
```go
type DateRangeList []DateRange
```
DateRangeList represents a list of DateRange values, it can be sorted and
searched using the slices package.

### Methods

```go
func (drl DateRangeList) Bound(year int, bound DateRange) DateRangeList
```
Bound returns a new list of date ranges that are bounded by the specified
date range.


```go
func (drl DateRangeList) Equal(year int, dr2 DateRangeList) bool
```
Equal returns true if the two DateRangeList values are equal for the given
year.


```go
func (drl DateRangeList) Merge(year int) DateRangeList
```
Merge returns a new list of date ranges that contains merged consecutive
overlapping ranges. The date list is assumed to be sorted.


```go
func (drl DateRangeList) MergeMonths(year int, months MonthList) DateRangeList
```
MergeMonths returns a merged list of date ranges that contains the specified
months for the given year.


```go
func (drl *DateRangeList) Parse(ranges []string) error
```
Parse ranges in formats '01:03', 'Jan:Mar', '01-02:03-04' or
'Jan-02:Mar-04'. The parsed list is sorted and without duplicates. If the
start date is identical then the end date is used to determine the order.


```go
func (drl DateRangeList) String() string
```




### Type DynamicDateRange
```go
type DynamicDateRange interface {
	Name() string
	Evaluate(year int) CalendarDateRange
}
```
DynamicDateRange is a function that returns a DateRange for a given year
and is intended to be evaluated once per year to calculate events such as
solstices, seasons or holidays.


### Type DynamicDateRangeList
```go
type DynamicDateRangeList []DynamicDateRange
```

### Methods

```go
func (dl DynamicDateRangeList) Evaluate(year int) []CalendarDateRange
```


```go
func (dl DynamicDateRangeList) String() string
```




### Type DynamicTimeOfDay
```go
type DynamicTimeOfDay interface {
	Name() string
	Evaluate(cd CalendarDate, yp Place) TimeOfDay
}
```
DynamicTimeOfDay is a function that returns a TimeOfDay for a given date
and is intended to be evaluated once per day to calculate events such as
sunrise, sunset etc.


### Type Month
```go
type Month uint8
```
Month as a uint8

### Constants
### January, February, March, April, May, June, July, August, September, October, November, December
```go
January Month = 1 + iota
February
March
April
May
June
July
August
September
October
November
December

```



### Functions

```go
func MirrorMonth(month Month) Month
```
MirrorMonth returns the month that is equidistant from the summer or winter
solstice for the specified month. For example, the mirror month for January
is November, and the mirror month for February is October.


```go
func ParseMonth(val string) (Month, error)
```
ParseMonth parses a month name of the form "Jan" to "Dec" or any other
longer prefixes of "January" to "December" in either lower or upper case.


```go
func ParseNumericMonth(val string) (Month, error)
```
ParseNumericMonth parses a 1 or 2 digit numeric month value in the range
1-12.



### Methods

```go
func (m Month) Days(year int) uint8
```


```go
func (m *Month) Parse(val string) error
```
Parse parses a month in either numeric or month name format.


```go
func (m Month) String() string
```




### Type MonthList
```go
type MonthList []Month
```
MonthList represents a list of Months, it can be sorted and searched the
slices package.

### Methods

```go
func (ml *MonthList) Parse(val string) error
```
Parse val in formats 'Jan,12,Nov'. The parsed list is sorted and without
duplicates.


```go
func (ml MonthList) String() string
```




### Type Place
```go
type Place struct {
	TimeLocation        *time.Location
	Latitude, Longitude float64
}
```
Place a location in terms of time.Location and a latitude and longitude.


### Type TimeOfDay
```go
type TimeOfDay uint32
```
TimeOfDay represents a time of day.

### Functions

```go
func NewTimeOfDay(hour, minute, second int) TimeOfDay
```
NewTimeOfDay creates a new TimeOfDay from the specified hour, minute and
second.


```go
func TimeOfDayFromTime(t time.Time) TimeOfDay
```
TimeOfDayFromTime returns a TimeOfDay from the specified time.Time.



### Methods

```go
func (t TimeOfDay) Add(delta time.Duration) TimeOfDay
```
Add delta to the time of day. The result will be normalized to 00:00:00 to
23:59:59.


```go
func (t TimeOfDay) Duration() time.Duration
```
Duration returns the time.Duration for the TimeOfDay.


```go
func (t TimeOfDay) Hour() int
```


```go
func (t TimeOfDay) Minute() int
```


```go
func (t TimeOfDay) Normalize() TimeOfDay
```
Normalize returns a new TimeOfDay with the hour, minute and second values
normalized to be within the valid range (0-23, 0-59, 0-59).


```go
func (t *TimeOfDay) Parse(val string) error
```
Parse val in formats '08[:12[:10]][am|pm]'


```go
func (t TimeOfDay) Second() int
```


```go
func (t TimeOfDay) String() string
```




### Type TimeOfDayList
```go
type TimeOfDayList []TimeOfDay
```

### Methods

```go
func (tl *TimeOfDayList) Parse(val string) error
```
Parse val as a comma separated list of TimeOfDay values.




### Type YearDay
```go
type YearDay uint32
```
YearDay represents a year and the day in that year as 1-365/366.

### Functions

```go
func NewYearDay(year, day int) YearDay
```
NewYearDay returns a YearDay for the given year and day. If the day is
greater than the number of days in the year then the last day of the year is
used.



### Methods

```go
func (yd YearDay) CalendarDate() CalendarDate
```


```go
func (yd YearDay) Date() Date
```
Date returns the Date for the given day of the year. A day of <= 0 is
treated as Jan-01 and a day of > 365/366 is treated as Dec-31.


```go
func (yd YearDay) Day() int
```


```go
func (yd YearDay) String() string
```


```go
func (yd YearDay) Year() int
```




### Type YearPlace
```go
type YearPlace struct {
	Year int
	Place
}
```
YearPlace represents a year at a given place.

### Functions

```go
func NewYearLocation(year int, loc *time.Location) YearPlace
```


```go
func NewYearPlace(year int, place Place) YearPlace
```


```go
func YearPlaceFromContext(ctx context.Context) YearPlace
```
YearPlaceFromContext returns the YearPlace value stored in the given
context, if there is no value stored then an empty YearPlace is returned for
which is IsNotset will be true.







### TODO
- cnicolaou: test with calendar dates also.
- cnicolaou: add calendar date ranges
- cnicolaou: add time ranges also.




