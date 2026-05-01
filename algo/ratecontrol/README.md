# Package [cloudeng.io/algo/ratecontrol](https://pkg.go.dev/cloudeng.io/algo/ratecontrol?tab=doc)

```go
import cloudeng.io/algo/ratecontrol
```

Package ratecontrol provides mechanisms for controlling the rate at which
requests are made and for backing off when the remote service is unwilling
to process requests.

## Constants
### DefaultTickInterval, DefaultRequestsPerTick, DefaultBytesPerTick, DefaultBackoffInterval, DefaultBackoffSteps
```go
DefaultTickInterval = time.Second
DefaultRequestsPerTick = 1
DefaultBytesPerTick = 1024 * 1024
DefaultBackoffInterval = time.Second
DefaultBackoffSteps = 10

```



## Types
### Type Backoff
```go
type Backoff interface {
	// Wait implements a backoff algorithm. It returns true if the backoff
	// should be terminated, i.e. no more requests should be attempted.
	// The error returned is nil when the backoff algorithm has reached
	// its limit and will generally only be non-nil for an internal error
	// such as the context being canceled.
	// The second argument is a placeholder for any additional data that
	// the backoff algorithm may need to process, such as an HTTP response
	// or a retry response. It can be nil if no such data is needed.
	Wait(context.Context, any) (bool, error)

	// Retries returns the number of retries that the backoff algorithm
	// has recorded, ie. the number of times that Backoff was called and
	// returned false.
	Retries() int
}
```
Backoff represents the interface to a backoff algorithm.


### Type Controller
```go
type Controller struct {
	// contains filtered or unexported fields
}
```
Controller implements Limiter and is used to control the rate at which
requests are made and to implement backoff when the remote server is
unwilling to process a request. Controller is safe to use concurrently.
Call Stop to free up resources when the Controller is no longer needed.
The controller attempts to implement a smooth rate of requests and bytes
over the specified tick intervals.

### Functions

```go
func New(opts ...Option) *Controller
```
New returns a new Controller configured using the specified options.



### Methods

```go
func (c *Controller) Backoff() Backoff
```
Backoff returns an instance of the configured backoff algorithm. If no
backoff algorithm is configured NoBackoff is returned.


```go
func (c *Controller) BytesTransferred(nBytes int)
```
BytesTransferred notifies the controller that the specified number of bytes
have been transferred and is used when byte based rate control is configured
via WithBytesPerTick.


```go
func (c *Controller) Stop()
```
Stop stops the Controller's tickers. It should be called when the Controller
is no longer needed to release resources.


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
ExponentialBackoff implements an exponential backoff algorithm. It starts
with the specified initial delay and doubles the delay for each retry up to
the specified number of steps.

### Functions

```go
func NewExponentialBackoff(initial time.Duration, steps int) *ExponentialBackoff
```
NewExponentialBackoff returns a instance of ExponentialBackoff. If initial
is less than or equal to zero, DefaultBackoffInterval is used. If steps is
less than or equal to zero, DefaultBackoffSteps is used.



### Methods

```go
func (eb *ExponentialBackoff) Retries() int
```
Retries implements Backoff.


```go
func (eb *ExponentialBackoff) Wait(ctx context.Context, _ any) (bool, error)
```
Wait implements Backoff.




### Type ExponentialBackoffOffset
```go
type ExponentialBackoffOffset struct {
	ExponentialBackoff
}
```
ExponentialBackoffOffset implements an exponential backoff algorithm with a
random offset used for the first delay, all subsequent delays are calculated
as in ExponentialBackoff. The first delay is a random value between 0 and
the initial delay.

### Functions

```go
func NewExponentialBackoffOffset(initial time.Duration, steps int) *ExponentialBackoffOffset
```
NewExponentialBackoffOffset returns a instance of ExponentialBackoffOffset.
If initial is less than or equal to zero, DefaultBackoffInterval is used.
If steps is less than or equal to zero, DefaultBackoffSteps is used.



### Methods

```go
func (eb *ExponentialBackoffOffset) Wait(ctx context.Context, v any) (bool, error)
```




### Type Limiter
```go
type Limiter interface {
	Wait(context.Context) error
	BytesTransferred(int)
	Backoff() Backoff
}
```
Limiter is an interface that defines a generic rate limiter.


### Type NoBackoff
```go
type NoBackoff struct{}
```
NoBackoff implements a Backoff that does not perform any backoff and always
returns false for Wait and 0 for Retries.

### Methods

```go
func (nb NoBackoff) Retries() int
```


```go
func (nb NoBackoff) Wait(_ context.Context, _ any) (bool, error)
```




### Type Option
```go
type Option func(c *options)
```
Option represents an option for configuring a ratecontrol Controller.

### Functions

```go
func WithBackoff(backoff func() Backoff) Option
```
WithBackoff allows the use of a custom backoff function.


```go
func WithBytesPerTick(tickInterval time.Duration, bpt int) Option
```
WithBytesPerTick sets the approximate rate in bytes per tick The algorithm
used is very simple and will simply stop sending data wait for a single tick
if the limit is reached without taking into account how long the tick is,
nor how much excess data was sent over the previous tick (ie. no attempt is
made to smooth out the rate and for now it's a simple start/stop model).
The bytes to be accounted for are reported to the Controller via its
BytesTransferred method. If tickInterval is less than or equal to zero,
DefaultTickInterval is used. If bpt is less than or equal to zero,
DefaultBytesPerTick is used.


```go
func WithExponentialBackoff(first time.Duration, steps int, randomizedOffset bool) Option
```
WithExponentialBackoff enables an exponential backoff algorithm.
If randomizedOffset is false NewExponentialBackoff is used, otherwise
NewExponentialBackoffOffset is used. If first is less than or equal to zero,
DefaultBackoffInterval is used. If steps is less than or equal to zero,
DefaultBackoffSteps is used.


```go
func WithNoRateControl() Option
```
WithNoRateControl creates a Controller that returns immediately and offers
no backoff. It can be used as a default when no rate control is desired.


```go
func WithRequestsPerTick(tickInterval time.Duration, rpt int) Option
```
WithRequestsPerTick sets the rate for requests in requests per tick.
If tickInterval is less than or equal to zero, DefaultTickInterval is used.
If rpt is less than or equal to zero, DefaultRequestsPerTick is used.







