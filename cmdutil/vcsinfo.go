// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"runtime/debug"
	"time"
)

// VCSInfo extracts version control system information from the build info
// if available. The returned values are the revision, last commit time,
// a boolean indicating whether there were uncommitted changes (dirty)
// and a boolean indicating whether the information was successfully extracted.
func VCSInfo() (revision string, lastCommit time.Time, dirty, ok bool) {
	var info *debug.BuildInfo
	info, ok = debug.ReadBuildInfo()
	if !ok {
		return
	}
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
