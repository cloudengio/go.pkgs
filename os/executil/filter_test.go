// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"

	"cloudeng.io/os/executil"
)

func Example() {
	ctx := context.Background()
	all := &bytes.Buffer{}
	cmd := exec.CommandContext(ctx, "cat", filepath.Join("testdata", "example"))
	ch := make(chan []byte, 1)
	filter := executil.NewLineFilter(all, regexp.MustCompile("filter me:"), ch)
	cmd.Stdout = filter
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { cmd.Start(); wg.Done() }()

	buf := <-ch
	fmt.Println("filtered output")
	fmt.Println(string(buf))
	fmt.Println("all of the output")
	fmt.Println(all.String())
	wg.Wait()
	if err := filter.Close(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// filtered output
	// filter me: 33
	// all of the output
	// some words
	// some more words
	// filter me: 33
	// and again more words
}
