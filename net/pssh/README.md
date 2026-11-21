# Package [cloudeng.io/net/pssh](https://pkg.go.dev/cloudeng.io/net/pssh?tab=doc)

```go
import cloudeng.io/net/pssh
```

Package pssh provides a persistent ssh connection, namely, one that will
be recreated if the connection is lost, over which commands can be run,
see Client.Run.

## Constants
### DefaultDialTimeout
```go
// DefaultDialTimeout is the default timeout for establishing a connection to
// the server. It is used if WithDialTimeout is not specified.
const DefaultDialTimeout = 30 * time.Second
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
  - A local forward whose port is already bound, which needs the port to be
    freed or the configuration changed.
  - Being refused permission by the local system.

DNS lookup failures (including "no such host", i.e. net.DNSError with
IsNotFound set) are treated as retryable: when a machine loses network or
VPN connectivity, OS resolvers routinely report EAI_NONAME ("no such host")
with IsTemporary and IsTimeout both false. Finite retry limits remain the
caller's bound for hostnames that genuinely do not exist.

Cancellation is not retryable either, but it is not a failed attempt:
err is context.Canceled or ErrClientClosed because the caller asked to stop.
The caller should test its own context rather than rely on this, and must
do so to distinguish a context whose deadline has expired -- reported here
as retryable, since a dial timeout is otherwise indistinguishable from it --
from one that is still live.

### Func ShellJoin
```go
func ShellJoin(args ...string) string
```
ShellJoin quotes each of args with ShellQuote and joins them with spaces,
to form a command line that runs args[0] with the remaining arguments.

### Func ShellQuote
```go
func ShellQuote(s string) string
```
ShellQuote quotes s so that a POSIX shell treats it as a single literal
word, with no expansion of any kind.



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
ConnectAndWait and terminated with Close.

### Functions

```go
func NewClient(ctx context.Context, network, addr string, opts ...Option) *Client
```
NewClient returns an instance of Client with the specified network and
address, and optional configuration options. A connection is not established
until ConnectAndWait is called.



### Methods

```go
func (c *Client) AuthKey() (ssh.PublicKey, string)
```
AuthKey returns the public key and comment (such as the key's filename or
identifier in the ssh agent) that was used to authenticate the current
or most recent connection. It returns nil, "" if no connection has
authenticated.


```go
func (c *Client) Close()
```


```go
func (c *Client) ConnectAndWait(ctx context.Context, backoff func() ratecontrol.Backoff) error
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


```go
func (c *Client) Run(ctx context.Context, cmd string, opts ...RunOption) error
```
Run runs cmd on the server, waiting first for the client to be connected.
The server runs cmd using the user's shell, as per ssh(1), and so it is
subject to that shell's quoting and expansion, and does not run in a
login shell unless WithLoginShell is used. Use ShellQuote and ShellJoin to
construct commands from arbitrary strings.

Run returns nil if the command exits with status 0, and an *ssh.ExitError,
which carries the status, if it does not; use errors.As to retrieve it.
If the connection is lost whilst the command is running the command
is not retried and the error is not an *ssh.ExitError, typically an
*ssh.ExitMissingError, since the command may have had some effect.
ErrClientClosed is returned if the client is closed.

If ctx is done the command is sent SIGTERM, which not all servers honour,
the session is closed and ctx.Err() is returned.

Many commands may be run concurrently, subject to the limit on sessions per
connection that the server imposes.


```go
func (c *Client) WaitConnected(ctx context.Context) error
```
WaitConnected waits until the client is connected, which is when
ConnectAndWait has established a connection. It returns ctx.Err() if ctx is
done first and ErrClientClosed if the client is closed.




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
func WithInterfaceRecovery(enable bool) Option
```
WithInterfaceRecovery enables retrying a connection that failed because
the route to the host could not reach it, by way of each local interface in
turn. See netutil.Dialer, which does the work and describes when it helps.

It is off by default, and is worth enabling only for a host on a subnet that
this system is attached to more than once, such as a machine on the same
LAN over both wired ethernet and wi-fi. Anywhere else a host that cannot be
reached is simply not reachable, and trying every interface only delays the
error that says so.


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
not specified, the logger from ctxlog.Logger is used.


```go
func WithOnAttemptFailed(fn func(err error, retryable bool)) Option
```
WithOnAttemptFailed registers a callback that is invoked each time an
attempt to establish the connection fails, with the error that it failed
with and whether retrying can resolve that error, as per Retryable. It is
not invoked when an attempt is abandoned because the context was cancelled
or the client was closed, since that is not a failure of the attempt.

It exists because an attempt that will be retried is otherwise not visible
to the caller: its error is not returned, as the connection is still being
maintained, and only the error from an attempt that cannot be retried is.
A caller that reports on a connection it is waiting for needs the former to
say why it is still waiting.

Note that retryable describes the error and not what will happen next: a
retryable error is the last one when the backoff allows no further attempt,
in which case ConnectAndWait returns it wrapped as an exhausted backoff.

Registering a callback takes over the reporting of these failures:
a retryable failure is logged only when there is no callback, so that it is
not reported twice.

The callback is invoked from the goroutine maintaining the connection and so
must not block.


```go
func WithOnConnect(fn func(*Client)) Option
```
WithOnConnect registers a callback that is invoked each time an ssh
connection and its configured port forwards have been established.


```go
func WithUser(user string) Option
```
WithUser specifies the user name used to authenticate to the server.
If it is not specified, the value of the USER environment variable is used.




### Type RunOption
```go
type RunOption func(*runOptions)
```
RunOption represents an option for Run.

### Functions

```go
func WithLoginShell() RunOption
```
WithLoginShell runs the command in the user's login shell, so that it sees
the environment, notably PATH, that the profile files set up. It is the
equivalent of running 'exec "$SHELL" -l -c cmd', which for some shells, eg.
zsh, reads the login but not the interactive startup files.


```go
func WithStderr(w io.Writer) RunOption
```
WithStderr specifies where the standard error of the command is written,
it is discarded if not specified.


```go
func WithStdin(r io.Reader) RunOption
```
WithStdin specifies the standard input of the command. It is discarded if
not specified. The reader must return, eg. when the context is cancelled,
for Run to do so.


```go
func WithStdout(w io.Writer) RunOption
```
WithStdout specifies where the standard output of the command is written,
it is discarded if not specified.







