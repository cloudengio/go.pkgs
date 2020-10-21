// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package userid provides analogous functionality to the standard os/user
// package except that it uses the 'id' command to obtain user and group
// information rather than /etc/passwd since on many system installations
// the user package will fail to find a user whereas the id command can.
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
	mu    sync.Mutex
	users map[string]IDInfo
}

// NewIDManager creates a new instance of IDManager.
func NewIDManager() *IDManager {
	return &IDManager{
		users: map[string]IDInfo{},
	}
}

func (idm *IDManager) exists(id string) (IDInfo, bool) {
	idm.mu.Lock()
	defer idm.mu.Unlock()
	info, ok := idm.users[id]
	return info, ok
}

// LookupID returns IDInfo for the specified user id or user name.
// It returns user.UnknownUserError if the user cannot be found or
// the invocation of the 'id' command fails somehow.
func (idm *IDManager) Lookup(id string) (IDInfo, error) {
	if id, exists := idm.exists(id); exists {
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
	idm.users[id] = info
	return info, nil
}

func runIDCommand(uid string) (string, error) {
	cmd := exec.Command("id", uid)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%v: %v", strings.Join(cmd.Args, " "), err)
	}
	return string(out), err
}
