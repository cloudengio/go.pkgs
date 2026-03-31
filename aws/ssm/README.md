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
goroutine so the caller is not blocked.

If PortForwardingSessionWithContext returns an error during the startup
window (before it has begun accepting connections), that error is returned
immediately. Once the session is forwarding, NewPortForwardingSession
returns a non-nil *Session; call Wait to block until the session ends and
retrieve any error that occurred during forwarding.

The session can be closed by canceling the supplied context.



### Methods

```go
func (s *Session) LocalPort() int
```
LocalPort returns the local port that is being forwarded to the remote host.
Clients can connect to this port to access the remote service through the
SSM tunnel.


```go
func (s *Session) Wait(duration time.Duration) error
```
Wait blocks until the session ends and returns any error that occurred
during forwarding. A nil error means the session was closed cleanly
(typically because the context was cancelled).







