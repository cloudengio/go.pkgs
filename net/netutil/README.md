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
is 443 then "https" will be appended to the address, otherwise the address
will be returned as is.

### Func ParseAddrDefaultPort
```go
func ParseAddrDefaultPort(addr, port string) (netip.AddrPort, error)
```
ParseAddrDefaultPort parses an IP address with an optional port, if the port
is specified the address it will be parsed as an address with a port and the
address will be returned, otherwise the default port will be appended to the
address and the address will be parsed as an address with that default port.

### Func ParseAddrIgnoringPort
```go
func ParseAddrIgnoringPort(addr string) (netip.Addr, error)
```
ParseAddrIgnoringPort parses an IP address string and returns the address.
If the string is an address with a port, it will be parsed as an address
with a port and the address will be returned, ignoring the port.

### Func ParseAddrOrPrefix
```go
func ParseAddrOrPrefix(addr string) (netip.Addr, error)
```
ParseAddrOrPrefix parses an IP address or prefix string and returns the
address. If the string is a prefix, it will be parsed as a prefix and the
address will be returned.




