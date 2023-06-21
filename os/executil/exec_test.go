// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"cloudeng.io/os/executil"
)

func ExampleCommand() {
	ctx := context.Background()
	tmpDir, _ := os.MkdirTemp("", "executil")
	execname := filepath.Join(tmpDir, "cat")
	execname, _ = executil.GoBuild(ctx, execname, filepath.Join("testdata", "cat.go"))
	cat := exec.Command(execname, filepath.Join("testdata", "example"))
	out, _ := cat.CombinedOutput()
	fmt.Println(string(out))
	// Output:
	// some words
	// some more words
	// filter me: 33
	// and again more words
}
