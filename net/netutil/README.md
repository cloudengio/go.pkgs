# Package [cloudeng.io/net/netutil](https://pkg.go.dev/cloudeng.io/net/netutil?tab=doc)

```go
import cloudeng.io/net/netutil
```

Package netutil provides utility functions for networking, including parsing
IP addresses and prefixes.

## Functions
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
ParseAddrDefaultPort calls Resolve to resolve the address before parsing it.

### Func ParseAddrIgnoringPort
```go
func ParseAddrIgnoringPort(addr string) (netip.Addr, error)
```
ParseAddrIgnoringPort parses an IP address string and returns the address.
If the string is an address with a port, it will be parsed as an
address with a port and the address will be returned, ignoring the port.
ParseAddrIgnoringPort calls Resolve to resolve the address before parsing
it.

### Func ParseAddrOrPrefix
```go
func ParseAddrOrPrefix(addr string) (netip.Prefix, error)
```
ParseAddrOrPrefix parses an IP address or prefix string and returns a
netip.Prefix. If the string is an IP address without a prefix, it is treated
as a full-bit prefix (/32 for IPv4, /128 for IPv6). ParseAddrOrPrefix calls
Resolve to resolve the address before parsing it.

### Func Resolve
```go
func Resolve(addr string) string
```
Resolve replaces the address component of addr with the first IP address
resolved for the host component of addr. If the host component of addr
cannot be resolved, addr is returned unchanged.




