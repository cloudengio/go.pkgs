// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build linux || darwin

package filewalk

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func dumpMemStats(size int) error {
	s := int(unsafe.Sizeof(Info{}))
	var rlim unix.Rlimit
	var rusage unix.Rusage
	if unix.Getrlimit(unix.RLIMIT_AS, &rlim) == nil && unix.Getrusage(0, &rusage) == nil {
		total := s * size
		fmt.Fprintf(os.Stderr, "rlimit current: [%v:%v] Info size %v bytes * %v = %v, max rss %v\n", rlim.Cur, rlim.Max, s, size, total, rusage.Maxrss)
		nt := (rusage.Maxrss * 1024) + int64(total)
		if nt > int64(rlim.Cur) {
			fmt.Fprintf(os.Stderr, "soft limit exceeded %v > %v\n", nt, int64(rlim.Cur))
			return fmt.Errorf("soft limit exceeded %v > %v", nt, int64(rlim.Cur))
		}
		if nt > int64(rlim.Max) {
			fmt.Fprintf(os.Stderr, "hard limit exceeded %v > %v\n", nt, int64(rlim.Max))
			return fmt.Errorf("hard limit exceeded %v > %v", nt, int64(rlim.Max))
		}
	}
	return nil
}
