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
	Wait(context.Context, *http.Response) (bool, error)

	// Retries returns the number of retries that the backoff aglorithm
	// has recorded, ie. the number of times that Backoff was called and
	// returned false.
	Retries() int
}
```
Backoff represents the interface to a backoff algorithm.

### Functions

```go
func NewExpontentialBackoff(initial time.Duration, steps int) Backoff
```
NewExpontentialBackoff returns a instance of Backoff that implements an
exponential backoff algorithm starting with the specified initial delay and
continuing for the specified number of steps.




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
func (eb *ExponentialBackoff) Wait(ctx context.Context, _ *http.Response) (bool, error)
```
Wait implements Backoff.




### Type Option
```go
type Option func(c *options)
```
Option represents an option for configuring a ratecontrol Controller.

### Functions

```go
func WithBytesPerTick(tickInterval time.Duration, bpt int) Option
```
The algorithm used is very simple and will simply stop sending data wait for
a single tick if the limit is reached without taking into account how long
the tick is, nor how much excess data was sent over the previous tick (ie.
no attempt is made to smooth out the rate and for now it's a simple
start/stop model). The bytes to be accounted for are reported to the
Controller via its BytesTransferred method.


```go
func WithCustomBackoff(backoff func() Backoff) Option
```


```go
func WithExponentialBackoff(first time.Duration, steps int) Option
```
WithExponentialBackoff enables an exponential backoff algorithm. First
defines the first backoff delay, which is then doubled for every consecutive
retry until the download either succeeds or the specified number of steps
(attempted requests) is exceeded.


```go
func WithRequestsPerTick(tickInterval time.Duration, rpt int) Option
```
WithRequestsPerTick sets the rate for requests in requests per tick.







