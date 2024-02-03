// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"bytes"
	"strings"
)

// called for urls of the form, but with the file:// prefix removed.
//
// Unix:
// file://localhost/etc/fstab
// file:///etc/fstab
//
// Windows:
// file://localhost/c:/WINDOWS/clock.avi
// file:///c:/WINDOWS/clock.avi
func parseFileURI(p string) (host, rest, drive string) {
	idx := strings.Index(p, "/")
	if idx < 0 {
		// file://xxx is not matched.
		return "", "", ""
	}
	host = p[:idx]
	rest = p[idx:]
	if d, ok := isWindowsDrive(p[idx+1:]); ok {
		rest = rest[1:]
		drive = d
	}
	return
}

// return the bucket and key from /bucket/key...
func bucketAndKey(path string, sep byte) (bucket, key string) {
	p := path
	switch len(path) {
	case 0:
		return
	case 1:
		if path[0] == sep {
			return
		}
	default:
		if path[0] == sep {
			p = p[1:]
		}
	}
	// p is now bucket/key...
	idx := bytes.Index([]byte(p), []byte{sep})
	if idx < 0 {
		bucket = p
		return
	}
	return p[:idx], p[idx+1:] // drop the leading sep from key.
}
