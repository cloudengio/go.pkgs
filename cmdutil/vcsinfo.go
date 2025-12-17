// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"encoding/json"
	"runtime/debug"
	"time"
)

// VCSInfo extracts version control system information from the build info
// if available. The returned values are the revision, last commit time,
// a boolean indicating whether there were uncommitted changes (dirty)
// and a boolean indicating whether the information was successfully extracted.
func VCSInfo() (goVersion, revision string, lastCommit time.Time, dirty, ok bool) {
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
func BuildInfoJSON() json.RawMessage {
	if bi, ok := debug.ReadBuildInfo(); ok {
		d, _ := json.Marshal(bi)
		return d
	}
	return nil
}
