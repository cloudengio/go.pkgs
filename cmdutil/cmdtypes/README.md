# Package [cloudeng.io/cmdutil/cmdtypes](https://pkg.go.dev/cloudeng.io/cmdutil/cmdtypes?tab=doc)

```go
import cloudeng.io/cmdutil/cmdtypes
```

Package cmdtypes provides types that are shared by the configuration and
command line packages, and that can be encoded and decoded as JSON, YAML and
text without those packages depending on each other.

## Constants
### Byte, KB, MB, GB, TB, KiB, MiB, GiB, TiB
```go
const (
	Byte ByteSize = 1

	KB ByteSize = 1_000
	MB          = 1_000 * KB
	GB          = 1_000 * MB
	TB          = 1_000 * GB

	KiB ByteSize = 1_024
	MiB          = 1_024 * KiB
	GiB          = 1_024 * MiB
	TiB          = 1_024 * GiB
)
```

### TimeSpanFormats
```go
const TimeSpanFormats = `a date and time such as 2024-03-05T10:30:00Z, "2024-03-05 10:30" or 10:30 (today), ` +
	`a date, month or year such as 2024-03-05, 2024-03 or 2024, ` +
	`or a time relative to now such as now, today, yesterday, -7d, now-1h or "2w ago" (units: ns us ms s m h d w)`
```
TimeSpanFormats describes what Span accepts, for use in usage messages.



## Variables
### FlexTimeFormats
```go
FlexTimeFormats = []string{
	time.RFC3339,
	time.DateTime,
	time.TimeOnly,
	time.DateOnly,
}

```
FlexTimeFormats are the layouts that a FlexTime is decoded from, tried
in this order. They are time.RFC3339, time.DateTime, time.TimeOnly and
time.DateOnly.



## Functions
### Func ParseDuration
```go
func ParseDuration(s string) (time.Duration, error)
```
ParseDuration is like time.ParseDuration, with the addition of the units d
(24 hours) and w (7 days). It does not accept a sign.



## Types
### Type ByteSize
```go
type ByteSize int64
```
ByteSize represents a quantity of bytes. It can be parsed from and
marshaled to human-readable strings using either binary (KiB, MiB, GiB,
TiB) or decimal (KB, MB, GB, TB) unit suffixes. A space between the number
and unit is optional; parsing is case-insensitive. Bare integers are treated
as bytes. Floating-point values are accepted during parsing (e.g. "1.5GiB").

Encoding and decoding is implemented by MarshalText and UnmarshalText,
which encoding/json, encoding/json/v2 and gopkg.in/yaml.v3 all use, so a
single type serves all three without this package depending on any of them.

### Functions

```go
func ParseByteSize(s string) (ByteSize, error)
```
ParseByteSize parses s into a ByteSize. Binary (KiB, MiB, GiB, TiB) and
decimal (KB, MB, GB, TB) suffixes are supported. A space between the number
and unit is allowed; parsing is case-insensitive. A bare number is treated
as bytes. Floating-point values are rounded to the nearest byte.



### Methods

```go
func (b ByteSize) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler, and with it the encoding used
for JSON and YAML.


```go
func (b ByteSize) String() string
```
String returns a human-readable representation of b. It selects the largest
decimal unit (TB, GB, MB, KB) that divides b evenly, then the largest binary
unit (TiB, GiB, MiB, KiB), and falls back to "NB" when no unit divides
evenly.


```go
func (b *ByteSize) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler, and with it the decoding
used for JSON and YAML.




### Type FlexTime
```go
type FlexTime time.Time
```
FlexTime is a time.Time that can be decoded from any of FlexTimeFormats and
is always encoded as time.RFC3339.

Encoding and decoding is implemented by MarshalText and UnmarshalText,
which encoding/json, encoding/json/v2 and gopkg.in/yaml.v3 all use, so a
single type serves all three without this package depending on any of them.

### Functions

```go
func ParseFlexTime(s string) (FlexTime, error)
```
ParseFlexTime parses s as any of FlexTimeFormats.



### Methods

```go
func (t FlexTime) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler, and with it the encoding used
for JSON and YAML.


```go
func (t FlexTime) String() string
```
String returns the time in time.RFC3339 format.


```go
func (t FlexTime) Time() time.Time
```
Time returns the time.Time represented by t.


```go
func (t *FlexTime) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler, and with it the decoding
used for JSON and YAML.




### Type Permissions
```go
type Permissions fs.FileMode
```
Permissions represents file permission bits (e.g. 0700, 0755).
It can be decoded from an octal number ("0700", "700", "0o700", "0x1c0"),
an rwx string ("rwxr-xr-x", "-rwx------", "rwx"), or symbolic chmod notation
("u=rwx,go=", "u=rwx,go=rx"), and is always encoded as the 4 digit octal
string, which every one of those formats can be read back from.

The setuid, setgid and sticky bits are represented as fs.ModeSetuid,
fs.ModeSetgid and fs.ModeSticky, whichever notation they were written in,
so that 4755 and rwsr-xr-x yield the same value. They are written out in the
traditional UNIX form, ie. in the leading octal digit.

Encoding and decoding is implemented by MarshalText and UnmarshalText, which
both encoding/json and gopkg.in/yaml.v3 use, so a single type serves both
without this package depending on either. YAML additionally routes unquoted
numbers such as 0700 through UnmarshalText, so they are read as octal rather
than as decimal. JSON has no octal literal, so a JSON number is handled by
UnmarshalJSON below, which reads it as octal for consistency.

### Functions

```go
func ParsePermissions(s string) (Permissions, error)
```
ParsePermissions parses a permission string in octal ("0700", "700",
"0o700", "0x1c0"), rwx ("rwxr-xr-x", "-rwx------", "rwx") or symbolic chmod
("u=rwx,go=", "u=rwx,go=rx") format.



### Methods

```go
func (p Permissions) FileMode() fs.FileMode
```
FileMode returns the fs.FileMode represented by p.


```go
func (p Permissions) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler, and with it the encoding used
for both JSON and YAML.


```go
func (p Permissions) String() string
```
String returns the 4 digit octal representation of the permissions (e.g.
"0700", "4755"), in the traditional UNIX form in which the leading digit
carries the setuid, setgid and sticky bits.


```go
func (p *Permissions) UnmarshalJSON(data []byte) error
```
UnmarshalJSON implements json.Unmarshaler. It exists only so that a JSON
number is read as octal, as it is in YAML and on a command line: JSON has no
octal literal, so 700 would otherwise be read as decimal and grant the wrong
bits. Strings are handled by UnmarshalText.


```go
func (p *Permissions) UnmarshalText(text []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler, and with it the decoding
used for both JSON strings and YAML scalars.




### Type TimeParser
```go
type TimeParser struct {
	// Location is the time zone of a value that does not give one, and the
	// zone that days, months and years begin in. It defaults to time.Local.
	Location *time.Location
	// Now returns the current time, which relative values are relative to.
	// It defaults to time.Now.
	Now func() time.Time
}
```
TimeParser parses the times and dates that a person writes on a command
line. Unlike FlexTime, which is read from and written to configuration
files and so is exact and unambiguous, it favours convenience: a value
may be coarse, "2024-03" for a month, or relative, "-7d" for a week ago,
and a value that gives no time zone is in Location rather than UTC.

The zero value uses time.Local and time.Now.

### Variables
### DefaultTimeParser
```go
DefaultTimeParser TimeParser

```
DefaultTimeParser is used by the flags that parse times. Tests may replace
it sequentially during setup to make relative values and zones repeatable;
it must not be modified concurrently with flag parsing.



### Methods

```go
func (p TimeParser) Parse(s string) (time.Time, error)
```
Parse returns the start of the span that s names, see Span.


```go
func (p TimeParser) Span(s string) (start, end time.Time, err error)
```
Span returns the span of time that s names, [start, end): a value names all
of the time to its precision, so that the date 2024-03-05 is the whole of
that day and the month 2024-03 the whole of that month, and a value to the
second is that second. start is the lower edge and end the upper, exclusive,
edge; a range of times uses the first of its beginning and the second of its
end, so that "2024-03-01..2024-03-05" includes all of the fifth.

A relative value is relative to the current time and is truncated to the
second, except expressions anchored at today, yesterday or tomorrow, which
preserve day-length spans (so "today+9h" spans 24 hours starting at 09:00).
A day is 24 hours and a week seven of them, whatever the clock does.







