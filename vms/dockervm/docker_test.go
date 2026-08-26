// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockervm_test

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"cloudeng.io/vms/dockervm"
)

func TestInspectContainerWithBinary_InvalidBinary(t *testing.T) {
	ctx := context.Background()
	_, _, err := dockervm.InspectContainerWithBinary(ctx, "nonexistent-binary-name-xyz", "test-container")
	if err == nil {
		t.Fatal("expected error executing nonexistent binary, got nil")
	}
	if _, ok := errors.AsType[*exec.Error](err); !ok {
		t.Logf("got error: %v", err)
	}
}
