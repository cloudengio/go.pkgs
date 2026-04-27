// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil_test

import (
	"time"

	"cloudeng.io/vms/vmstestutil"
)

func init() {
	vmstestutil.SetTestConfig(vmstestutil.PoolTestConfig{
		Constructor:     vmstestutil.NewMockFactory(),
		PoolSize:        2,
		SupportsSuspend: true,
		Timeout:         10 * time.Second,
	})
}
