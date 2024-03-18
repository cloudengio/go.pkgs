// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
)

func main() {
	for i, a := range os.Args[1:] {
		if v := os.Getenv(a); v != "" {
			fmt.Print(v)
		} else {
			fmt.Print(a)
		}
		if i < len(os.Args)-1 {
			fmt.Print(" ")
		}
	}
	fmt.Println()
}
