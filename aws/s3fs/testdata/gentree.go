// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import "fmt"

func main() {
	for i := 0; i < 5; i++ {
		for j := 0; j < 10; j++ {
			for k := 0; k < 100; k++ {
				fmt.Printf("bucket/tb/p1%03v/p2%03v/p3%03v\n", i, j, k)
			}
		}
	}
}
