// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"cloudeng.io/os/executil"
)

func Example() {
	go func() {
		time.Sleep(time.Second * 5)
		buf := make([]byte, 1024*1024)
		n := runtime.Stack(buf, true)
		fmt.Fprintf(os.Stderr, "%s\n", string(buf[:n]))
		os.Exit(1)
	}()
	ctx := context.Background()
	all := &bytes.Buffer{}
	// Use go run testdata/cat.go for compatibility across windows and unix.
	cmd := exec.CommandContext(ctx, "go", "run", filepath.Join("testdata", "cat.go"), filepath.Join("testdata", "example")) // #nosec G204
	ch := make(chan []byte, 1)
	filter := executil.NewLineFilter(all, ch, regexp.MustCompile("filter me:"))
	cmd.Stdout = filter
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := cmd.Start(); err != nil {
			panic(err)
		}
	})

	buf := <-ch
	fmt.Println("filtered output")
	fmt.Println(string(buf))
	fmt.Println("all of the output")
	wg.Wait()
	if err := filter.Close(); err != nil {
		fmt.Println(err)
	}
	fmt.Println(all.String())

	// Output:
	// filtered output
	// filter me: 33
	// all of the output
	// some words
	// some more words
	// filter me: 33
	// and again more words
}
