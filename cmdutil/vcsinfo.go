// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"runtime/debug"
	"time"
)

func VCSInfo() (revision string, lastCommit time.Time, dirty, ok bool) {
	var info *debug.BuildInfo
	info, ok = debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			revision = kv.Value
		case "vcs.time":
			lastCommit, _ = time.Parse(time.RFC3339, kv.Value)
		case "vcs.modified":
			dirty = kv.Value == "true"
		}
	}
	return
}
