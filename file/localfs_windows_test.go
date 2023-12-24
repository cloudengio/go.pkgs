// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// go:build windows
package file

import (
	"context"
	"testing"

	"cloudeng.io/file"
)

func TestMergeXAttr(t *testing.T) {
	ctx := context.Background()
	x := file.XAttr{
		UID:       1,
		GID:       2,
		Device:    3,
		FileID:    4,
		Blocks:    5,
		Hardlinks: 6,
	}
	fs := file.LocalFS()
	xattr := fs.SysXAttr(nil, x)
	stat, ok := xattr.(*sysinfo)
	if !ok {
		t.Fatalf("got %T, want *sysinfo", xattr)
	}
	nxattr := fs.SysXAttr(sysinfo, x)
	if got, want:nxattr, xattr; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
