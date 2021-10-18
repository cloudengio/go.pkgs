// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package userid provides complimentary functionality to the standard os/user
// package by using the 'id' command to avoid loss of functionality when
// cross compiling. It provides minimal functionality for windows, see below.
// On Unix systems, it first uses the os/user package and then falls back to the
// using the 'id' command. It offers reduced functionality as compared to
// os/user. By way of background os/user has both a pure-go implementation and
// a cgo implementation. The former parses /etc/passwd and the latter uses the
// getwpent operations. The cgo implementation cannot be used when cross
// compiling since cgo is generally disabled for cross compilation.
// Hence applications that use os/user can find themselves losing the ability
// to resolve info for all users when cross compiled and used on systems that
// use a directory service that is accessible via getpwent but whose members
// do not appear in the text file /etc/passwd.
//
// For windows it uses the PowerShell to obtain minimal information on
// the user and windows SID and represents that information in the same
// format as the 'id' command.
package userid

import (
	"fmt"
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
	users  map[string]IDInfo     // keyed by username or uid
	groups map[string]user.Group // keyed by groupname or gid
}

// NewIDManager creates a new instance of IDManager.
func NewIDManager() *IDManager {
	return &IDManager{
		users:  map[string]IDInfo{},
		groups: map[string]user.Group{},
	}
}

func (idm *IDManager) userExists(id string) (IDInfo, bool) {
	idm.mu.Lock()
	defer idm.mu.Unlock()
	info, ok := idm.users[id]
	return info, ok
}

func (idm *IDManager) groupExists(id string) (user.Group, bool) {
	idm.mu.Lock()
	defer idm.mu.Unlock()
	grp, ok := idm.groups[id]
	return grp, ok
}

func infoUsingIDCommand(id string) (IDInfo, error) {
	out, err := runIDCommand(id)
	if err != nil {
		return IDInfo{}, user.UnknownUserError(id)
	}
	info, err := ParseIDCommandOutput(out)
	if err != nil {
		return IDInfo{}, user.UnknownUserError(id)
	}
	return info, nil
}

func infoUsingUserPackage(id string) (IDInfo, error) {
	u, err := user.LookupId(id)
	if err != nil {
		u, err = user.Lookup(id)
	}
	if err == nil {
		info := IDInfo{
			UID:      u.Uid,
			Username: usernameOnly(u.Username),
		}
		grp, err := user.LookupGroupId(u.Gid)
		if err == nil {
			info.Groups = []user.Group{*grp}
		}
		return info, nil
	}
	return IDInfo{}, user.UnknownUserError(id)
}

func (idm *IDManager) update(info IDInfo) {
	idm.mu.Lock()
	defer idm.mu.Unlock()
	// Save the same information for both user and groups.
	idm.users[info.Username] = info
	idm.users[info.UID] = info
	for _, grp := range info.Groups {
		idm.groups[grp.Name] = grp
		idm.groups[grp.Gid] = grp
	}
}

// LookupUser returns IDInfo for the specified user id or user name.
// It returns user.UnknownUserError if the user cannot be found or
// the invocation of the 'id' command fails somehow.
func (idm *IDManager) LookupUser(id string) (IDInfo, error) {
	if id, exists := idm.userExists(id); exists {
		return id, nil
	}
	info, err := infoUsingUserPackage(id)
	if err == nil {
		idm.update(info)
		return info, nil
	}
	info, err = infoUsingIDCommand(id)
	if err != nil {
		return info, err
	}
	idm.update(info)
	return info, nil
}

// LookupGroup returns IDInfo for the specified group id or group name.
// It returns user.UnknownGroupError if the group cannot be found or
// the invocation of the 'id' command fails somehow.
func (idm *IDManager) LookupGroup(id string) (user.Group, error) {
	if grp, exists := idm.groupExists(id); exists {
		return grp, nil
	}
	grp, err := user.LookupGroupId(id)
	if err != nil {
		grp, err = user.LookupGroup(id)
	}
	if err == nil {
		idm.mu.Lock()
		defer idm.mu.Unlock()
		idm.groups[grp.Name] = *grp
		idm.groups[grp.Gid] = *grp
		return *grp, nil
	}

	// run id for the current user in the hope that it discovers the
	// group.
	idm.LookupUser("")
	if id, exists := idm.groupExists(id); exists {
		return id, nil
	}
	return user.Group{}, user.UnknownGroupError(id)
}
