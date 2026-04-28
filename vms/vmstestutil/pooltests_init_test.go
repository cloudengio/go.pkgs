// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil_test

import (
	"time"

	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

//go:generate astest --preamble=cfg=testConfig . pooltests_test.go

var testConfig = vmstestutil.PoolTestConfig{
	Constructor:      vmstestutil.NewMockFactory(true),
	PoolSize:         2,
	StagingBehaviour: vmspool.StagingBehaviourSuspended,
	Timeout:          10 * time.Second,
}
