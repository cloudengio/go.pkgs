// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil

import (
	"context"
	"io"
	"sync"
	"time"

	"cloudeng.io/vms"
)

// Mock represents a mock virtual machine instance for testing.
type Mock struct {
	mu         sync.Mutex
	state      vms.State
	properties vms.Properties
	isSuspend  bool

	CloneErr   error
	StartErr   error
	StopRunErr error
	StopErr    error
	SuspendErr error
	DeleteErr  error
	ExecErr    error
}

// NewMock creates a new Mock VM instance.
func NewMock() *Mock {
	return &Mock{
		state:     vms.StateInitial,
		isSuspend: true,
	}
}

func (m *Mock) Clone(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CloneErr != nil {
		return m.CloneErr
	}
	m.state = vms.StateStopped
	return nil
}

func (m *Mock) Start(ctx context.Context, stdout, stderr io.Writer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StartErr != nil {
		m.state = vms.StateErrorUnknown
		return m.StartErr
	}
	m.state = vms.StateRunning
	return nil
}

func (m *Mock) Stop(ctx context.Context, timeout time.Duration) (error, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StopRunErr != nil || m.StopErr != nil {
		m.state = vms.StateErrorUnknown
		return m.StopRunErr, m.StopErr
	}
	m.state = vms.StateStopped
	return nil, nil
}

func (m *Mock) Suspendable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isSuspend
}

func (m *Mock) SetSuspendable(suspendable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isSuspend = suspendable
}

func (m *Mock) Suspend(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SuspendErr != nil {
		m.state = vms.StateErrorUnknown
		return m.SuspendErr
	}
	m.state = vms.StateSuspended
	return nil
}

func (m *Mock) Delete(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.state = vms.StateDeleted
	return nil
}

func (m *Mock) State(ctx context.Context) vms.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Mock) SetState(state vms.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

func (m *Mock) Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error {
	if m.ExecErr != nil {
		return m.ExecErr
	}
	return nil
}

func (m *Mock) Properties(ctx context.Context) (vms.Properties, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.properties, nil
}

func (m *Mock) SetProperties(props vms.Properties) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.properties = props
}

var _ vms.Instance = (*Mock)(nil)
