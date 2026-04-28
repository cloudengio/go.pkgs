// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil_test

import (
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/vmstestutil"
)

//go:generate astest --match='^TestInstance' -preamble=cfg=instanceTestConfig . instancetests_test.go

var instanceTestConfig = vmstestutil.InstanceTestConfig{
	Constructor: func() vms.Instance { return vmstestutil.NewMock("test-instance") },
	Timeout:     10 * time.Second,
	ExecCmd:     "echo",
	ExecArgs:    []string{"hello"},
}
