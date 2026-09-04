// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// VCSInfo extracts version control system information from the build info,
// if available. It returns, in order:
//
//   - goVersion: the version of Go the executable was built with.
//   - revision: the vcs.revision recorded at build time.
//   - lastCommit: the vcs.time recorded at build time, ie. the time of that
//     revision.
//   - execModTime: the modification time of the executable file. This is an
//     approximation of when the executable was built and no more than that:
//     it is the file's mtime, so it changes if the file is copied, touched or
//     unpacked from an archive, and it is unrelated to the build info.
//   - dirty: whether vcs.modified was recorded, ie. whether the tree had
//     uncommitted changes when it was built.
//   - ok: whether any vcs setting was found.
//
// ok reports only on the vcs settings. goVersion and execModTime are
// determined independently of them and may be set even when ok is false, as
// happens for an executable built from a directory that is not a repository,
// or with -buildvcs=false. The zero values of revision, lastCommit and dirty
// are not distinguishable from values that were genuinely absent, so ok is
// what should be tested before reporting them.
func VCSInfo() (goVersion, revision string, lastCommit, execModTime time.Time, dirty, ok bool) {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if fi, err := os.Stat(exe); err == nil {
			execModTime = fi.ModTime().UTC()
		}
	}
	var info *debug.BuildInfo
	info, ok = debug.ReadBuildInfo()
	if !ok {
		return
	}
	goVersion = info.GoVersion
	ok = false
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			revision = kv.Value
			ok = true
		case "vcs.time":
			lastCommit, _ = time.Parse(time.RFC3339, kv.Value)
			ok = true
		case "vcs.modified":
			dirty = kv.Value == "true"
			ok = true
		}
	}
	return
}

// BuildInfoJSON returns the build information as a JSON raw message
// or nil if the build information is not available.
func BuildInfoJSON() jsontext.Value {
	if bi, ok := debug.ReadBuildInfo(); ok {
		d, _ := json.Marshal(bi)
		return d
	}
	return nil
}
