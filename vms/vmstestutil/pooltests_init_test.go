// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil_test

import (
	"time"

	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

<<<<<<< ours
<<<<<<< New base: .
//go:generate astest --match='^TestPool' --preamble=cfg=testConfig . pooltests_test.go

var testConfig = vmstestutil.PoolTestConfig{
	Constructor:      vmstestutil.NewMockFactory(true),
	PoolSize:         2,
	StagingBehaviour: vmspool.StagingBehaviourSuspended,
	Timeout:          10 * time.Second,
||||||| Common ancestor
func init() {
	vmstestutil.SetTestConfig(vmstestutil.PoolTestConfig{
		Constructor:     vmstestutil.NewMockFactory(),
		PoolSize:        2,
		SupportsSuspend: true,
		Timeout:         10 * time.Second,
	})
=======
//go:generate astest --preamble=cfg=testConfig . pooltests_test.go
||||||| ancestor
func init() {
	vmstestutil.SetTestConfig(vmstestutil.PoolTestConfig{
		Constructor:     vmstestutil.NewMockFactory(),
		PoolSize:        2,
		SupportsSuspend: true,
		Timeout:         10 * time.Second,
	})
||||||| Common ancestor
func init() {
	vmstestutil.SetTestConfig(vmstestutil.PoolTestConfig{
		Constructor:     vmstestutil.NewMockFactory(),
		PoolSize:        2,
		SupportsSuspend: true,
		Timeout:         10 * time.Second,
	})
=======
//go:generate astest --preamble=cfg=testConfig . pooltests_test.go
=======
//go:generate astest --match='^TestPool' --preamble=cfg=testConfig . pooltests_test.go
>>>>>>> theirs

var testConfig = vmstestutil.PoolTestConfig{
	Constructor:      vmstestutil.NewMockFactory(true),
	PoolSize:         2,
	StagingBehaviour: vmspool.StagingBehaviourSuspended,
	Timeout:          10 * time.Second,
}
