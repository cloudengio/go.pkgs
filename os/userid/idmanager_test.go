// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package userid

import (
	"os"
	"testing"
)

func TestManager(t *testing.T) {
	idm := NewIDManager()
	user := os.Getenv("USER")
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
}
