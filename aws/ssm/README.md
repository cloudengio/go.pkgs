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
NewPortForwardingSession starts a new SSM port forwarding session based on
the provided input parameters. It returns a Session object that can be used
to retrieve the local port for the forwarded connection. The session can be
closed by canceling the supplied context.



### Methods

```go
func (s *Session) LocalPort() int
```
LocalPort returns the local port that is being forwarded to the remote host.
Clients can connect to this port to access the remote service through the
SSM tunnel.







