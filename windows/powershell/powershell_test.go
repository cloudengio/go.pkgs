// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build windows

package powershell_test

import (
	"os"
	"strings"
	"testing"

	"cloudeng.io/windows/powershell"
)

func TestSimple(t *testing.T) {
	ps := powershell.New()
	stdout, stderr, err := ps.Run(`$env:username`)
	if err != nil {
		t.Fatalf("failed: %v %v", stderr, err)
	}
	stdout = strings.TrimSpace(stdout)
	t.Log(stdout)
	if len(stdout) < 2 {
		t.Errorf("looks too small to be a valid user name)")
	}
	if os.Getenv("CIRCLECI") == "true" {
		if got, want := stdout, "circleci"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
