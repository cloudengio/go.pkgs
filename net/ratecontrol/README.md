# Package [cloudeng.io/net/ratecontrol](https://pkg.go.dev/cloudeng.io/net/ratecontrol?tab=doc)

```go
import cloudeng.io/net/ratecontrol
```

Package ratecontrol provides mechanisms for controlling the rate at which
requests are made and for backing off when the remote service is unwilling
to process requests.

Package ratecontrol provides mechanisms for controlling the rate at which
requests are made and for implementing backoff mechanisms.

## Types
### Type Backoff
```go
type Backoff interface {
	// Wait implements a backoff algorithm. It returns true if the backoff
	// should be terminated, i.e. no more requests should be attempted.
	// The error returned is nil when the backoff algorithm has reached
	// its limit and will generally only be non-nil for an internal error
	// such as the context being cancelled.
	Wait(context.Context) (bool, error)

	// Retries returns the number of retries that the backoff aglorithm
	// has recorded, ie. the number of times that Backoff was called and
	// returned false.
	Retries() int
}
```
Backoff represents the interface to a backoff algorithm.

### Functions

```go
func NewExpontentialBackoff(clock Clock, initial time.Duration, steps int) Backoff
```
NewExpontentialBackoff returns a instance of Backoff that implements an
exponential backoff algorithm starting with the specified initial delay and
continuing for the specified number of steps.




### Type Clock
```go
type Clock interface {
	Tick() int
	TickDuration() time.Duration
	// contains filtered or unexported methods
}
```
Clock represents a clock used for rate limiting. It determines the current
time in 'ticks' and the wall-clock duration of a tick. This allows for rates
to be specified in terms of requests per tick or bytes per tick rather than
over a fixed duration. A default Clock implementation is provided which uses
time.Minute as the tick duration.


### Type Controller
```go
type Controller struct {
	// contains filtered or unexported fields
}
```
Controller is used to control the rate at which requests are made and to
implement backoff when the remote server is unwilling to process a request.
Controller is safe to use concurrently.

### Functions

```go
func New(opts ...Option) *Controller
```
New returns a new Controller configuring using the specified options.



### Methods

```go
func (c *Controller) Backoff() Backoff
```


```go
func (c *Controller) BytesTransferred(nBytes int)
```
BytesTransferred notifies the controller that the specified number of bytes
have been transferred and is used when byte based rate control is configured
via WithBytesPerTick.


```go
func (c *Controller) Wait(ctx context.Context) error
```
Wait returns when a request can be made. Rate limiting of requests takes
priority over rate limiting of bytes. That is, bytes are only considered
when a new request can be made.




### Type ExponentialBackoff
```go
type ExponentialBackoff struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (eb *ExponentialBackoff) Retries() int
```
Retries implements Backoff.


```go
func (eb *ExponentialBackoff) Wait(ctx context.Context) (bool, error)
```
Wait implements Backoff.




### Type HourClock
```go
type HourClock struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (c HourClock) Tick() int
```


```go
func (c HourClock) TickDuration() time.Duration
```




### Type MinuteClock
```go
type MinuteClock struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (c MinuteClock) Tick() int
```


```go
func (c MinuteClock) TickDuration() time.Duration
```




### Type Option
```go
type Option func(c *options)
```
Option represents an option for configuring a ratecontrol Controller.

### Functions

```go
func WithBytesPerTick(bpt int) Option
```
WithBytesPerTick sets the approximate rate in bytes per tick, where tick is
the unit of time reported by the Clock implementation in use. The default
clock uses time.Now().Minute() and hence the rate is in bytes per minute.
The algorithm used is very simple and will wait for a single tick if the
limit is reached without taking into account how long the tick is, nor how
much excess data was sent over the previous tick (ie. no attempt is made to
smooth out the rate and for now it's a simple start/stop model). The bytes
to be accounted for are reported to the Controller via its BytesTransferred
method.


```go
func WithClock(c Clock) Option
```
WithClock sets the clock implementation to use.


```go
func WithExponentialBackoff(first time.Duration, steps int) Option
```
WithExponentialBackoff enables an exponential backoff algorithm. First
defines the first backoff delay, which is then doubled for every consecutive
retry until the download either succeeds or the specified number of steps
(attempted requests) is exceeded.


```go
func WithRequestsPerTick(rpt int) Option
```
WithRequestsPerTick sets the rate for requests in requests per tick,
where tick is the unit of time reported by the Clock implementation in use.
The default clock uses time.Now().Minute() as the interval for rate limiting
and hence the rate is in requests per minute.




### Type SecondClock
```go
type SecondClock struct {
	// contains filtered or unexported fields
}
```

### Methods

```go
func (c SecondClock) Tick() int
```


```go
func (c SecondClock) TickDuration() time.Duration
```







