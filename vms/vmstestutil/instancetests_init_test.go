// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil_test

import (
	"context"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmstestutil"
)

//go:generate astest --match='^TestInstance' -preamble=cfg=instanceTestConfig . instancetests_test.go

var instanceTestConfig = vmstestutil.InstanceTestConfig{
	Constructor: vmstestutil.NewMockFactory(true),
	Timeout:     10 * time.Second,
	ExecCmd:     "echo",
	ExecArgs:    []string{"hello"},
	RequireUnderlyingState: func(ctx context.Context, inst vms.Instance, msg string, final vms.State, intermediate ...vms.State) error {
		// no point in testing the underlying state of the mock.
		return vms.WaitForState(ctx, inst, time.Millisecond, final, intermediate...)
	},
}
