// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"testing"
	"time"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
	"cloudeng.io/file/matcher"
)

func TestUserGroup(t *testing.T) {
	uid := matcher.NewUser("user", func(name string) (uint64, error) {
		return 100, nil
	})
	gid := matcher.NewUser("group", func(name string) (uint64, error) {
		return 300, nil
	})
	fi := file.NewInfo("foo", 0, 0, time.Time{}, file.XAttr{UID: 100, GID: 300})

	expr, err := boolexpr.New(boolexpr.NewOperandItem(uid))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := expr.Eval(fi), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	expr, err = boolexpr.New(boolexpr.NewOperandItem(gid))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := expr.Eval(fi), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
