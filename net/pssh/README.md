# Package [cloudeng.io/net/pssh](https://pkg.go.dev/cloudeng.io/net/pssh?tab=doc)

```go
import cloudeng.io/net/pssh
```

Package pssh provides a persistent ssh connection, namely, one that can will
be recreated if the connection is lost.

## Constants
### DefaultDialTimeout
```go
// DefaultDialTimeout is the default timeout for establishing a connection to
// the server. It is used if WithDialTimeout is not specified.
DefaultDialTimeout = 30 * time.Second

```



## Variables
### ErrNoAgent, ErrKnownHosts, ErrClientClosed
```go
// ErrNoAgent is returned when no ssh agent is available at all.
ErrNoAgent = errors.New("no ssh agent available")
// ErrKnownHosts is returned when the known hosts files cannot be used,
// either because none of them exist or because they cannot be parsed.
ErrKnownHosts = errors.New("no usable known hosts file")
// ErrClientClosed is returned when the client has been closed.
ErrClientClosed = errors.New("client closed")

```
Errors that a retry cannot resolve, since they describe how the client is
configured rather than the state of the network or the server.



## Functions
### Func Retryable
```go
func Retryable(err error) bool
```
Retryable reports whether a connection attempt that failed with err may
succeed if it is attempted again, as opposed to one that will fail the same
way however often it is repeated.

Errors are treated as retryable unless they are known not to be.
A persistent client exists to stay connected, so an unrecognised failure
should not end it; the backoff bounds how long an error that is in fact
permanent is retried for. The cases below are therefore the exceptions.

Not retryable:

  - Anything to do with the identity of the server. A key that does not
    match the one recorded may be an interception attempt, and reconnecting
    to a host that presents it is exactly what must not happen. An unknown
    host stays unknown, and a revoked key stays revoked.
  - Authentication failure. A retry offers the server the same keys,
    and repeating it counts against MaxAuthTries and any lockout policy.
  - This package's own configuration errors: ErrNoAgent, ErrKnownHosts.
  - Algorithm negotiation failure. The two ends do not have a cipher, MAC,
    key exchange or host key type in common, and will not acquire one.
  - Malformed networks and addresses, which are in the configuration.
  - A hostname that does not exist. Note that this is distinguished from a
    DNS server that could not be reached, which is retryable.
  - A local forward whose port is already bound, which needs the port to be
    freed or the configuration changed.
  - Being refused permission by the local system.

Cancellation is not retryable either, but it is not a failed attempt:
err is context.Canceled or ErrClientClosed because the caller asked to stop.
The caller should test its own context rather than rely on this, and must
do so to distinguish a context whose deadline has expired -- reported here
as retryable, since a dial timeout is otherwise indistinguishable from it --
from one that is still live.



## Types
### Type Client
```go
type Client struct {
	// contains filtered or unexported fields
}
```
Client represents a persistent ssh connection to a server. It is created
with NewClient, which takes the network and address of the server,
and optional configuration. The persistent connection is established with
Connect and terminated with Close.

### Functions

```go
func NewClient(ctx context.Context, network, addr string, opts ...Option) *Client
```
NewClient returns an instance of Client with the specified network and
address, and optional configuration options. A connection is not established
until Connect is called.



### Methods

```go
func (c *Client) Close()
```


```go
func (c *Client) ConnectAndWait(ctx context.Context, backoff func() ratecontrol.Backoff) (net.Conn, error)
```
ConnectAndWait establishes a connection to the server, along with any
configured port forwards, and maintains it: whenever the connection is lost
it is established again. It does not return once the connection is up, but
only when it stops being maintained, which is when the context is cancelled,
when the client is closed, or when an attempt fails for a reason that
retrying cannot resolve. The error returned says which; Retryable describes
how that last case is determined.

backoff is called to obtain the delays between attempts, and is called again
after each successful connection so that the delays accumulated reaching
a server do not carry over into the next outage. A backoff that gives up
bounds how long a connection is retried for, and ConnectAndWait then returns
the error from the last attempt.




### Type Option
```go
type Option func(*options)
```

### Functions

```go
func WithAcceptNewHostKeys(accept bool) Option
```
WithAcceptNewHostKeys determines what happens when the server is not listed
in any known hosts file. The default, and the behaviour when accept is
false, is to refuse the connection: there is no user to prompt in the way
that ssh(1) does, and a client that reconnects indefinitely must not accept
a new key on every reconnection.

When accept is true a host that is not yet known is trusted and recorded,
which is the behaviour of ssh(1)'s StrictHostKeyChecking=accept-new.
A host that is already known but presents a different key is still refused:
that is the case this option is not intended to relax.


```go
func WithDialTimeout(timeout time.Duration) Option
```
WithDialTimeout specifies the timeout for establishing a connection to the
server. The default is DefaultDialTimeout.


```go
func WithHostKey(key ssh.PublicKey) Option
```
WithHostKey pins the public key that the server is required to present,
bypassing the known hosts files entirely. It takes precedence over
WithKnownHosts.


```go
func WithKnownHosts(files ...string) Option
```
WithKnownHosts specifies the known hosts files used to verify the server,
in the format used by ssh(1). If no files are named, or this option is not
used at all, ~/.ssh/known_hosts and /etc/ssh/ssh_known_hosts are used.
Named files that do not exist are ignored, as ssh(1) ignores them.


```go
func WithLocalPortForward(localPort, remotePort int) Option
```
WithLocalPortForward specifies a port forward from localPort to remotePort.
It can be used multiple times to specify multiple forwards.


```go
func WithLogger(logger *slog.Logger) Option
```
WithLogger specifies the logger used to report connection events. If it is
not specified, the logger the ctxlog.Logger is used.


```go
func WithUser(user string) Option
```
WithUser specifies the user name used to authenticate to the server.
If it is not specified, the value of the USER environment variable is used.







