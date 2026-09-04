// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
	"time"

	"cloudeng.io/cmdutil"
)

// execModTime determines the modification time of the running test binary in
// the same way that VCSInfo does, so that it can be used as an independent
// expectation for the value VCSInfo returns. It reports false if the time
// cannot be obtained, which VCSInfo treats as best effort and this test
// therefore has nothing to compare against.
func execModTime() (time.Time, bool) {
	exe, err := os.Executable()
	if err != nil {
		return time.Time{}, false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	fi, err := os.Stat(exe)
	if err != nil {
		return time.Time{}, false
	}
	return fi.ModTime().UTC(), true
}

// TestVCSInfo verifies what each result depends on. A test binary is normally
// built without vcs settings, so this exercises the case where ok is false
// while the values that do not come from those settings are still present.
func TestVCSInfo(t *testing.T) {
	goVersion, revision, lastCommit, modTime, dirty, ok := cmdutil.VCSInfo()
	t.Logf("goVersion=%q revision=%q lastCommit=%v execModTime=%v dirty=%v ok=%v",
		goVersion, revision, lastCommit, modTime, dirty, ok)

	// The modification time comes from the executable file rather than from
	// the build info, so it is set whether or not vcs settings were found. It
	// is best effort however, and is left as the zero time in environments
	// where the executable cannot be located or stat'ed.
	if want, avail := execModTime(); !avail {
		t.Log("skipping the executable modification time: it cannot be obtained in this environment")
	} else if !modTime.Equal(want) {
		t.Errorf("execModTime: got %v, want %v", modTime, want)
	}

	// The Go version comes from the build info, not the vcs settings, so it
	// too is independent of ok. It is similarly best effort: it is empty if
	// the build info is unavailable.
	bi, avail := debug.ReadBuildInfo()
	if !avail {
		t.Log("skipping the go version and ok: build info is unavailable in this environment")
		return
	}
	if goVersion != bi.GoVersion {
		t.Errorf("goVersion: got %q, want %q (with ok=%v)", goVersion, bi.GoVersion, ok)
	}

	// ok must reflect the presence of the vcs settings and nothing else, so
	// determine independently whether this binary was built with any of them.
	// Both outcomes occur in practice: test binaries are usually built without
	// them, but a build with -buildvcs does record them.
	wantOK := false
	for _, kv := range bi.Settings {
		switch kv.Key {
		case "vcs.revision", "vcs.time", "vcs.modified":
			wantOK = true
		}
	}
	if ok != wantOK {
		t.Errorf("ok: got %v, want %v for settings %v", ok, wantOK, bi.Settings)
	}

	// When there are no vcs settings none of the values derived from them
	// should be set: their zero values would otherwise be reported as though
	// they were real.
	if !ok {
		if revision != "" {
			t.Errorf("ok is false but revision is %q", revision)
		}
		if !lastCommit.IsZero() {
			t.Errorf("ok is false but lastCommit is %v", lastCommit)
		}
		if dirty {
			t.Error("ok is false but dirty is true")
		}
	}
}
