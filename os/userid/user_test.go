// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package userid_test

import (
	"os/user"
	"reflect"
	"testing"

	"cloudeng.io/os/userid"
)

func TestParse(t *testing.T) {
	ida := userid.IDInfo{
		UID:       "384864",
		Username:  "user",
		GID:       "8577",
		Groupname: "group",
		Groups: []user.Group{
			{Gid: "1", Name: "g1"},
		},
	}
	id, err := userid.ParseIDCommandOutput("uid=384864(user) gid=8577(group) groups=1(g1)")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := id, ida; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	ida.Groups = []user.Group{
		{Gid: "1", Name: "g1"},
		{Gid: "22", Name: "g2"},
		{Gid: "3791", Name: "g3"},
	}
	id, err = userid.ParseIDCommandOutput("uid=384864(user) gid=8577(group) groups=1(g1),22(g2),3791(g3)")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := id, ida; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseWindowsUserids(t *testing.T) {
	s := func(a ...string) []string {
		return a
	}
	for _, tc := range []struct {
		sid       string
		authority string
		sub       []string
	}{
		{"S-1-0-0", "0", s("0")},
		{"S-1-5-32-544", "5", s("32", "544")},
		{"S-1-5-21-255908664-2662632750-4148280483-500", "5", s("21", "255908664", "2662632750", "4148280483", "500")},
	} {
		v, a, sa := userid.ParseWindowsSID(tc.sid)
		if got, want := v, "1"; got != want {
			t.Errorf("%v: got %v, want %v", tc.sid, got, want)
		}
		if got, want := a, tc.authority; got != want {
			t.Errorf("%v: got %v, want %v", tc.sid, got, want)
		}
		if got, want := sa, tc.sub; !reflect.DeepEqual(got, want) {
			t.Errorf("%v: got %v, want %v", tc.sid, got, want)
		}
	}

	d, u := userid.ParseWindowsUser(`domain\user`)
	if got, want := d, "domain"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := u, "user"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
