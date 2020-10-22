// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package userid provides complimentary functionality to the standard os/user
// package by using the 'id' command to avoid loss of functionality when
// cross compiling. By way of background os/user has both a pure-go
// implementation and a cgo implementation. The former parses /etc/passwd
// and the latter uses the getwpent operations. The cgo implementation
// cannot be used when cross compiling since cgo is generally disabled for
// cross compilation. Hence applications that use os/user can find themselves
// losing the ability to resolve info for all users when cross compiled and used
// on systems that use a directory service that is accessible via getpwent
// but whose members do not appear in the text file /etc/passwd.
package userid

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
	"sync"
)

// IDInfo represents the parsed output of the 'id' command.
type IDInfo struct {
	UID, Username  string
	GID, Groupname string
	Groups         []user.Group
}

// parse a(b) and return a, b
func parseItem(item string) (id, name string) {
	idxA := strings.Index(item, "(")
	idxB := strings.Index(item, ")")
	if idxA < 0 || idxB < 0 || idxA >= idxB {
		return
	}
	return item[:idxA], item[idxA+1 : idxB]
}

// parse x=y and return x, y
func parseEquals(field string) (x, y string) {
	parts := strings.Split(field, "=")
	if len(parts) != 2 {
		return
	}
	return parts[0], parts[1]
}

// ParseIDCommandOutput parses the output of the unix id command.
func ParseIDCommandOutput(out string) (IDInfo, error) {
	var id IDInfo
	parts := strings.Split(out, " ")
	if got, want := len(parts), 3; got != want {
		return id, fmt.Errorf("wrong # of space separated fields, got %v, not %v", got, want)
	}
	x, y := parseEquals(parts[0])
	if x == "uid" {
		id.UID, id.Username = parseItem(y)
	} else {
		return id, fmt.Errorf("first field is not the uid field: %v", x)
	}
	x, y = parseEquals(parts[1])
	if x == "gid" {
		id.GID, id.Groupname = parseItem(y)
	} else {
		return id, fmt.Errorf("second field is not the gid field: %v", x)
	}
	x, y = parseEquals(parts[2])
	if x == "groups" {
		for _, grp := range strings.Split(y, ",") {
			gid, name := parseItem(grp)
			id.Groups = append(id.Groups, user.Group{Gid: gid, Name: name})
		}
	} else {
		return id, fmt.Errorf("third field is not the groups field: %v", x)
	}
	return id, nil
}

// IDManager implements a caching lookup of user information by
// id or username that uses the 'id' command.
type IDManager struct {
	mu     sync.Mutex
	users  map[string]IDInfo // keyed by username or uid
	groups map[string]IDInfo // keyed by groupname or gid
}

// NewIDManager creates a new instance of IDManager.
func NewIDManager() *IDManager {
	return &IDManager{
		users:  map[string]IDInfo{},
		groups: map[string]IDInfo{},
	}
}

func (idm *IDManager) userExists(id string) (IDInfo, bool) {
	idm.mu.Lock()
	defer idm.mu.Unlock()
	info, ok := idm.users[id]
	return info, ok
}

func (idm *IDManager) groupExists(id string) (IDInfo, bool) {
	idm.mu.Lock()
	defer idm.mu.Unlock()
	info, ok := idm.groups[id]
	return info, ok
}

// LookupUser returns IDInfo for the specified user id or user name.
// It returns user.UnknownUserError if the user cannot be found or
// the invocation of the 'id' command fails somehow.
func (idm *IDManager) LookupUser(id string) (IDInfo, error) {
	if id, exists := idm.userExists(id); exists {
		return id, nil
	}
	out, err := runIDCommand(id)
	if err != nil {
		return IDInfo{}, user.UnknownUserError(id)
	}
	info, err := ParseIDCommandOutput(out)
	if err != nil {
		return IDInfo{}, user.UnknownUserError(id)
	}
	idm.mu.Lock()
	defer idm.mu.Unlock()
	// Save the same information for both user and groups.
	idm.users[info.Username] = info
	idm.users[info.UID] = info
	for _, grp := range info.Groups {
		idm.groups[grp.Name] = info
		idm.groups[grp.Gid] = info
	}
	return info, nil
}

// LookupGroup returns IDInfo for the specified group id or group name.
// It returns user.UnknownGroupError if the group cannot be found or
// the invocation of the 'id' command fails somehow.
func (idm *IDManager) LookupGroup(id string) (IDInfo, error) {
	if id, exists := idm.groupExists(id); exists {
		return id, nil
	}
	// run id for the current user in the hope that it discovers the
	// group.
	idm.LookupUser("")
	if id, exists := idm.groupExists(id); exists {
		return id, nil
	}
	return IDInfo{}, user.UnknownGroupError(id)
}

func runIDCommand(uid string) (string, error) {
	args := []string{}
	if len(uid) > 0 {
		args = append(args, uid)
	}
	cmd := exec.Command("id", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%v: %v", strings.Join(cmd.Args, " "), err)
	}
	return string(out), err
}
