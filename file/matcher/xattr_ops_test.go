// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package matcher_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"cloudeng.io/cmdutil/boolexpr"
	"cloudeng.io/file"
	"cloudeng.io/file/matcher"
	"cloudeng.io/os/userid"
)

type withXattr struct {
	x file.XAttr
}

func (w withXattr) XAttr() file.XAttr {
	return w.x
}

func TestUserGroup(t *testing.T) {
	uid := matcher.NewUser("user", "100", func(text string) (file.XAttr, error) {
		id, _ := strconv.ParseInt(text, 10, 32)
		return file.XAttr{UID: id}, nil
	})
	gid := matcher.NewGroup("group", "300", func(text string) (file.XAttr, error) {
		id, _ := strconv.ParseInt(text, 10, 32)
		return file.XAttr{GID: id}, nil
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

func TestUserGroupParsing(t *testing.T) {
	ctx := context.Background()
	tmpdir := t.TempDir()
	filename := filepath.Join(tmpdir, "file1")
	if err := os.WriteFile(filename, []byte{'a', 'b'}, 0600); err != nil {
		t.Fatal(err)
	}

	// Test a file in a temp dir and a locally existing one since on windows
	// the file owners are different.
	for _, testfile := range []string{filename, "xattr_ops_test.go"} {
		fs := file.LocalFS()
		info, err := fs.Stat(ctx, testfile)
		if err != nil {
			t.Fatal(err)
		}
		xattr, err := fs.XAttr(ctx, testfile, info)
		var fileUID, fileGID string
		if xattr.UID != -1 {
			fileUID = fmt.Sprintf("%v", xattr.UID)
		} else {
			fileUID = xattr.User
		}
		if xattr.GID != -1 {
			fileGID = fmt.Sprintf("%v", xattr.GID)
		} else {
			fileGID = xattr.Group
		}

		idm := userid.NewIDManager()
		uid := matcher.NewUser("user", fileUID, func(text string) (file.XAttr, error) {
			return matcher.ParseUsernameOrID(text, idm.LookupUser)
		})
		gid := matcher.NewGroup("group", fileGID, func(text string) (file.XAttr, error) {
			return matcher.ParseGroupnameOrID(text, idm.LookupGroup)
		})

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
}
