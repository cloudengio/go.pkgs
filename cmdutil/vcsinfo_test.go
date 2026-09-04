// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"testing"

	"cloudeng.io/cmdutil"
)

func TestVCSInfo(t *testing.T) {
	goVersion, revision, lastCommit, buildTime, dirty, ok := cmdutil.VCSInfo()
	if buildTime.IsZero() {
		t.Errorf("expected non-zero buildTime from os.Executable()")
	}
	t.Logf("goVersion=%q revision=%q lastCommit=%v buildTime=%v dirty=%v ok=%v",
		goVersion, revision, lastCommit, buildTime, dirty, ok)
}
