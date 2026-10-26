# Package [cloudeng.io/net/netutil](https://pkg.go.dev/cloudeng.io/net/netutil?tab=doc)

```go
import cloudeng.io/net/netutil
```

Package netutil provides utility functions for networking, including parsing
IP addresses and prefixes.

## Functions
### Func EnsureHostPort
```go
func EnsureHostPort(addr, port string) string
```
EnsureHostPort returns addr that is guaranteed to have a port. If addr
already has a port, it is returned unchanged. If addr does not have a port,
the supplied port is appended. If addr is an IPv6 address, it will be
enclosed in brackets if it is not already.

### Func HTTPServerAddr
```go
func HTTPServerAddr(addrPort netip.AddrPort) string
```
HTTPServerAddr returns the address of an HTTP server based on the address
and port of the server in a form to be used with http.Server.Addr.
If the address is unspecified then it will be replaced with an empty string.
If the port is 80 then "http" will be appended to the address, if the port
is 443 then "https" will be appended to the address, otherwise the numeric
port will be used.

### Func ParseAddrDefaultPort
```go
func ParseAddrDefaultPort(addr, defaultPort string) (netip.AddrPort, error)
```
ParseAddrDefaultPort parses an IP address string. If the address
string already contains a port, it is parsed and returned. Otherwise,
the supplied default port is used to construct and parse an address with
that port. If the address contains only a port an address of "::" is used.
ParseAddrDefaultPort calls ResolveFirst to resolve the address before
parsing it.

### Func ParseAddrIgnoringPort
```go
func ParseAddrIgnoringPort(addr string) (netip.Addr, error)
```
ParseAddrIgnoringPort parses an IP address string and returns the address.
If the string is an address with a port, it will be parsed as an
address with a port and the address will be returned, ignoring the port.
ParseAddrIgnoringPort calls ResolveFirst to resolve the address before
parsing it.

### Func ParseAddrOrPrefix
```go
func ParseAddrOrPrefix(addr string) (netip.Prefix, error)
```
ParseAddrOrPrefix parses an IP address or prefix string and returns a
netip.Prefix. If the string is an IP address without a prefix, it is treated
as a full-bit prefix (/32 for IPv4, /128 for IPv6). ParseAddrOrPrefix calls
ResolveFirst to resolve the address before parsing it.

### Func Resolve
```go
func Resolve(addr string) string
```
Resolve replaces the host component of addr with the first IP address
resolved for that host. If the host component of addr cannot be resolved,
addr is returned unchanged.

Deprecated: Use ResolveFirst instead, which returns an error if the host
cannot be resolved instead of returning the original address.

### Func ResolveAll
```go
func ResolveAll(addr string) ([]net.IP, error)
```
ResolveAll returns all IP addresses resolved for the host component of addr.

### Func ResolveFirst
```go
func ResolveFirst(addr string) (string, error)
```
ResolveFirst returns addr with the host component replaced by the first IP
address resolved for that host; any port is preserved unchanged. If the host
component is empty (e.g. "" or ":80") addr is returned unchanged since there
is nothing to resolve. If the host cannot be resolved, an error is returned.

### Func TrimIPv6
```go
func TrimIPv6(addr string) string
```
TrimIPv6 removes brackets from an IPv6 address if they are present.



## Types
### Type Dialer
```go
type Dialer struct {
	net.Dialer

	// RecoverNoRouteToHost enables retrying a connection that failed with
	// EHOSTUNREACH by way of each local interface in turn, as described
	// above. It is off by default.
	RecoverNoRouteToHost bool
}
```
Dialer is a net.Dialer that recovers from a host which is unreachable over
the route the system chooses for it, but reachable over another, by retrying
the connection from each local interface in turn.

It exists for a host on a subnet that the system is attached to more than
once, such as a machine on the same LAN over both wired ethernet and wi-fi.
Each attachment contributes its own route to that subnet and a connection
goes out of whichever of them the kernel selects. When it selects one
whose neighbour discovery for the target has not succeeded, the connection
fails at once with EHOSTUNREACH, "no route to host", even though the
host is reachable over the other. Which route is selected is not stable,
so the same address can be reachable from one process, or at one moment,
and not from the next.

Dial and DialContext first dial exactly as the embedded net.Dialer does.
Only if that fails with EHOSTUNREACH do they try again, binding each attempt
to a different local address, which selects the route by source address. The
first connection to succeed is returned. If none does, the error from the
original attempt is returned rather than the last one, since it describes
the path the system was configured to use.

The recovery is opt-in: RecoverNoRouteToHost must be set for any of it to
happen. A host that cannot be reached is usually simply not reachable,
and trying every interface then only delays the error and obscures it;
the recovery is for a system attached to one subnet more than once,
which the caller knows about and this type cannot detect. The zero value is
a usable Dialer that dials exactly as its embedded net.Dialer does.

A Dialer with its LocalAddr set is left alone even when the recovery is
enabled: a caller that has chosen a source address has already said which
path to use, so there is nothing to recover from.

Timeout bounds the original attempt and then, separately, all of the retries
together, so a dial takes at most about twice it rather than it once per
local address. Deadline, being absolute, bounds everything as it would for
a net.Dialer. Note that a route which cannot reach the host fails at once,
without consuming either, which is why cycling through the interfaces of a
multihomed system is cheap.

Only EHOSTUNREACH is recovered from. It means the route that was used cannot
reach the host, which is what another route may fix. A refused connection or
a timeout says the route reached something, and trying from elsewhere would
only obscure that.

### Methods

```go
func (d *Dialer) Dial(network, address string) (net.Conn, error)
```
Dial connects to the address on the named network, as per net.Dialer.Dial,
recovering from EHOSTUNREACH as described for Dialer when
RecoverNoRouteToHost is set.


```go
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error)
```
DialContext connects to the address on the named network, as per
net.Dialer.DialContext, recovering from EHOSTUNREACH as described for Dialer
when RecoverNoRouteToHost is set.







