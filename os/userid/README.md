# Package [cloudeng.io/os/userid](https://pkg.go.dev/cloudeng.io/os/userid?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/os/userid)](https://goreportcard.com/report/cloudeng.io/os/userid)

```go
import cloudeng.io/os/userid
```

Package userid provides analogous functionality to the standard os/user
package except that it uses the 'id' command to obtain user and group
information rather than /etc/passwd since on many system installations the
user package will fail to find a user whereas the id command can.

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
func (idm *IDManager) Lookup(id string) (IDInfo, error)
```
LookupID returns IDInfo for the specified user id or user name. It returns
user.UnknownUserError if the user cannot be found or the invocation of the
'id' command fails somehow.







