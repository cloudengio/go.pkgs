// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"os"
)

func main() {
	for _, file := range os.Args[1:] {
		rd, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, rd)
	}
}
