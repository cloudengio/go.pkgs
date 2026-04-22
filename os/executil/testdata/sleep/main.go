// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}

	f, err := strconv.ParseFloat(os.Args[1], 64)
	if err != nil {
		os.Exit(1)
	}

	time.Sleep(time.Duration(f * float64(time.Second)))
}
