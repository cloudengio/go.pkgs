// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"strconv"
	"testing"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
	"cloudeng.io/file/matcher"
)

type withXattr struct {
	x file.XAttr
}

func (w withXattr) XAttr() file.XAttr {
	return w.x
}

func TestUserGroup(t *testing.T) {
	uid := matcher.NewUser("user", "100", func(name string) (uint64, error) {
		return strconv.ParseUint(name, 10, 32)
	})
	gid := matcher.NewGroup("group", "300", func(name string) (uint64, error) {
		return strconv.ParseUint(name, 10, 32)
	})
	xattr := file.XAttr{UID: 100, GID: 300}
	wx := withXattr{xattr}

	expr, err := boolexpr.New(boolexpr.NewOperandItem(uid))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := expr.Eval(wx), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	expr, err = boolexpr.New(boolexpr.NewOperandItem(gid))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := expr.Eval(wx), true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
