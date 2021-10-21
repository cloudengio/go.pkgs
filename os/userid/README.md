# Package [cloudeng.io/os/userid](https://pkg.go.dev/cloudeng.io/os/userid?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/os/userid)](https://goreportcard.com/report/cloudeng.io/os/userid)

```go
import cloudeng.io/os/userid
```

Package userid provides complimentary functionality to the standard os/user
package by using the 'id' command to avoid loss of functionality when cross
compiling. It provides minimal functionality for windows, see below. On Unix
systems, it first uses the os/user package and then falls back to the using
the 'id' command. It offers reduced functionality as compared to os/user. By
way of background os/user has both a pure-go implementation and a cgo
implementation. The former parses /etc/passwd and the latter uses the
getwpent operations. The cgo implementation cannot be used when cross
compiling since cgo is generally disabled for cross compilation. Hence
applications that use os/user can find themselves losing the ability to
resolve info for all users when cross compiled and used on systems that use
a directory service that is accessible via getpwent but whose members do not
appear in the text file /etc/passwd.

For windows it uses the PowerShell to obtain minimal information on the user
and windows SID and represents that information in the same format as the
'id' command.

## Functions
### Func GetCurrentUser
```go
func GetCurrentUser() string
```
GetCurrentUser returns the current user as determined by environment
variables.

### Func ParseWindowsSID
```go
func ParseWindowsSID(sid string) (version, auth string, sub []string)
```
ParseWindowsSID parses a windows Security Identifier (SID).

### Func ParseWindowsUser
```go
func ParseWindowsUser(u string) (domain, user string)
```
ParseWindowsUser returns the domain and user component of a windows username
(domain\user).



## Types
### Type IDInfo
```go
type IDInfo struct {
	UID, Username  string
	GID, Groupname string
	Groups         []user.Group
}
```
IDInfo represents the parsed output of the 'id' command.

### Functions

```go
func ParseIDCommandOutput(out string) (IDInfo, error)
```
ParseIDCommandOutput parses the output of the unix id command.




### Type IDManager
```go
type IDManager struct {
	// contains filtered or unexported fields
}
```
IDManager implements a caching lookup of user information by id or username
that uses the 'id' command.

### Functions

```go
func NewIDManager() *IDManager
```
NewIDManager creates a new instance of IDManager.



### Methods

```go
func (idm *IDManager) LookupGroup(id string) (user.Group, error)
```
LookupGroup returns IDInfo for the specified group id or group name. It
returns user.UnknownGroupError if the group cannot be found or the
invocation of the 'id' command fails somehow.


```go
func (idm *IDManager) LookupUser(id string) (IDInfo, error)
```
LookupUser returns IDInfo for the specified user id or user name. It returns
user.UnknownUserError if the user cannot be found or the invocation of the
'id' command fails somehow.







