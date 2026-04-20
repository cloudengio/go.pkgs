# Package [cloudeng.io/aws/ssm](https://pkg.go.dev/cloudeng.io/aws/ssm?tab=doc)

```go
import cloudeng.io/aws/ssm
```


## Types
### Type Session
```go
type Session struct {
	// contains filtered or unexported fields
}
```
Session represents an active SSM port forwarding session. It provides access
to the local port that is being forwarded to the remote host.

### Functions

```go
func NewPortForwardingSession(ctx context.Context, pfi ssmclient.PortForwardingInput) (*Session, error)
```
NewPortForwardingSession starts a new SSM port forwarding session based
on the provided input parameters. The underlying forwarding call runs in a
goroutine so the caller is not blocked. Note that the tunnel will be ready
to accept connections when NewPortForwardingSession returns. If LocalPort
is not specified in the input, a free local port will be automatically
allocated and used for the session. The caller can retrieve the local port
being used via the LocalPort method.

The session can be closed by canceling the supplied context.



### Methods

```go
func (s *Session) LocalPort() int
```
LocalPort returns the local port that is being forwarded to the remote host.
Clients can connect to this port to access the remote service through the
SSM tunnel.


```go
func (s *Session) Wait(ctx context.Context, duration time.Duration) error
```
Wait blocks until the session ends and returns any error that occurred
during forwarding. A nil error means the session was closed cleanly
(typically because the context was cancelled).







