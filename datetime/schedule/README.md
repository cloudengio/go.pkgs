# Package [cloudeng.io/datetime/schedule](https://pkg.go.dev/cloudeng.io/datetime/schedule?tab=doc)

```go
import cloudeng.io/datetime/schedule
```

Package schedule provides support for scheduling events based on dates and
times.

## Types
### Type ActionSpec
```go
type ActionSpec[T any] struct {
	Name    string
	Due     datetime.TimeOfDay
	Dynamic DynamicTimeOfDaySpec
	Repeat  RepeatSpec
	T       T
}
```
ActionSpec represents a specification of an action to be taken at specific
times of that day. The specification may refer to a dynamically evaluated
time of day (eg. sunrise/sunset) with an offset. In addition it may specify
a repeat interval and an optional number of times for the action to be
repeated.

### Methods

```go
func (a ActionSpec[T]) Evaluate(cd datetime.CalendarDate, place datetime.Place) ActionSpec[T]
```
Evaluate returns a new ActionSpec with the Due field set to the result of
evaluating the DynamicDue field if it is non-nil (with the DynamicOffset
applied).




### Type ActionSpecs
```go
type ActionSpecs[T any] []ActionSpec[T]
```

### Methods

```go
func (a ActionSpecs[T]) Evaluate(cd datetime.CalendarDate, place datetime.Place) ActionSpecs[T]
```
Evaluate returns a new ActionSpecs with each of the ActionSpecs evaluated,
that is, with any Dynamic functions evaluated and the results stored in the
Due field. The returned ActionSpecs have their Dynamic fields zeroed out.


```go
func (a ActionSpecs[T]) Sort()
```
Sort by due time and then by name.


```go
func (a ActionSpecs[T]) SortStable()
```
Sort by due time, but preserve the order of actions with the same due time.




### Type Active
```go
type Active[T any] struct {
	Name string
	When time.Time
	T    T
}
```
Active represents the next scheduled action, ie. the one to be 'active' at
time 'When'.


### Type AnnualScheduler
```go
type AnnualScheduler[T any] struct {
	// contains filtered or unexported fields
}
```
AnnualScheduler provides a way to iterate over the specified actions for a
single year.

### Functions

```go
func NewAnnualScheduler[T any](actions ActionSpecs[T]) *AnnualScheduler[T]
```
NewAnnualScheduler returns a new annual scheduler with the supplied
schedule.



### Methods

```go
func (s *AnnualScheduler[T]) Scheduled(yp datetime.YearPlace, dates Dates, bounds datetime.DateRange) iter.Seq[Scheduled[T]]
```
Scheduled returns an iterator over the scheduled actions for the given
year and place that returns all of the scheduled actions for each day that
has scheduled Actions. It will evaluate any dynamic due times and sort the
actions by their evaluated due time.




### Type Dates
```go
type Dates struct {
	Months       datetime.MonthList            // Whole months to include.
	MirrorMonths bool                          // Include the 'mirror' months of those in For.
	Ranges       datetime.DateRangeList        // Include specific date ranges.
	Dynamic      datetime.DynamicDateRangeList // Functions to generate dates that vary by year, such as solstices, seasons or holidays.
	Constraints  datetime.Constraints          // Constraints to be applied, such as weekdays/weekends etc.
}
```
Dates represents a set of dates expressed as a combination of months,
date ranges and constraints on those dates (eg. weekdays in March).

### Methods

```go
func (d Dates) EvaluateDateRanges(year int, bounds datetime.DateRange) datetime.DateRangeList
```
EvaluateDateRanges returns the list of date ranges that are represented by
the totality of the information represented by Dates instance, including the
evaluation of dynamic date ranges. The result is bounded by supplied bounds
date range.


```go
func (d Dates) String() string
```




### Type DynamicTimeOfDaySpec
```go
type DynamicTimeOfDaySpec struct {
	Due    datetime.DynamicTimeOfDay
	Offset time.Duration
}
```
DynamicTimeOfDaySpec represents a time of day that is dynamically evaluated
and offset with a fixed duration.


### Type RepeatSpec
```go
type RepeatSpec struct {
	Interval time.Duration
	Repeats  int
}
```
RepeatSpec represents a repeat interval and an optional number of times for
an action to be repeated.


### Type Scheduled
```go
type Scheduled[T any] struct {
	Date  datetime.CalendarDate
	Specs ActionSpecs[T]
}
```
Scheduled specifies the set of actions scheduled for a given date.

### Methods

```go
func (s Scheduled[T]) Active(place datetime.Place) iter.Seq[Active[T]]
```
Active is an iterator that returns the next scheduled action for the given
year and place.







