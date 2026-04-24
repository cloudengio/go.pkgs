// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil_test

import (
	"testing"
	"time"

	"cloudeng.io/vms/vmstestutil"
)

func TestRunPoolTests(t *testing.T) {
	f := vmstestutil.NewMockFactory()
	vmstestutil.RunPoolTests(t, vmstestutil.PoolTestConfig{
		Constructor:     f,
		PoolSize:        2,
		SupportsSuspend: true,
		Timeout:         10 * time.Second,
	})
}
