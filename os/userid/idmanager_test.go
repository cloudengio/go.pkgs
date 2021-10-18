// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package userid

import (
	"testing"
)

func TestManager(t *testing.T) {
	idm := NewIDManager()
	user := GetCurrentUser()
	id, err := idm.LookupUser(user)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := id.Username, user; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	id, ok := idm.userExists(user)
	if got, want := id.Username, user; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := ok, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	id, ok = idm.userExists(id.UID)
	if got, want := ok, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := id.Username, user; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	idm.mu.Lock()
	defer idm.mu.Unlock()
	if len(idm.groups) == 0 {
		t.Errorf("groups should not be empty")
	}
}
