# Package [cloudeng.io/net/ratecontrol](https://pkg.go.dev/cloudeng.io/net/ratecontrol?tab=doc)

```go
import cloudeng.io/net/ratecontrol
```

Package ratecontrol provides mechanisms for controlling the rate at
which requests are made and for backing off when the remote service is
unwilling to process requests. DEPRECATED: This package has been moved to
cloudeng.io/algo/ratecontrol; use that instead.

## Functions
### Func NewExpontentialBackoff
```go
func NewExpontentialBackoff(initial time.Duration, steps int) ratecontrol.Backoff
```
NewExpontentialBackoff returns a instance of Backoff that implements an
exponential backoff algorithm starting with the specified initial delay and
continuing for the specified number of steps.



## Types
### Type Backoff
```go
type Backoff ratecontrol.Backoff
```


### Type Controller
```go
type Controller struct {
	*ratecontrol.Controller
}
```
Controller implements Limiter and is used to control the rate at which
requests are made and to implement backoff when the remote server is
unwilling to process a request. Controller is safe to use concurrently.

### Functions

```go
func New(opts ...Option) *Controller
```
New returns a new Controller configuring using the specified options.




### Type Limiter
```go
type Limiter ratecontrol.Limiter
```


### Type Option
```go
type Option ratecontrol.Option
```
Option represents an option for configuring a ratecontrol Controller.

### Functions

```go
func WithBytesPerTick(tickInterval time.Duration, bpt int) Option
```
WithBytesPerTick sets the approximate rate in bytes per tick The algorithm
used is very simple and will simply stop sending data wait for a single tick
if the limit is reached without taking into account how long the tick is,
nor how much excess data was sent over the previous tick (ie. no attempt is
made to smooth out the rate and for now it's a simple start/stop model).
The bytes to be accounted for are reported to the Controller via its
BytesTransferred method.


```go
func WithCustomBackoff(backoff func() Backoff) Option
```
WithCustomBackoff allows the use of a custom backoff function.


```go
func WithExponentialBackoff(first time.Duration, steps int) Option
```
WithExponentialBackoff enables an exponential backoff algorithm. First
defines the first backoff delay, which is then doubled for every consecutive
retry until the download either succeeds or the specified number of steps
(attempted requests) is exceeded.


```go
func WithNoRateControl() Option
```
WithNoRateControl creates a Controller that returns immediately and offers
no backoff. It can be used as a default when no rate control is desired.


```go
func WithRequestsPerTick(tickInterval time.Duration, rpt int) Option
```
WithRequestsPerTick sets the rate for requests in requests per tick.







