// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmstestutil

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"cloudeng.io/vms"
)

// ExecCall records a single invocation of Mock.Exec.
type ExecCall struct {
	Cmd  string
	Args []string
}

// Mock represents a mock virtual machine instance for testing.
type Mock struct {
	id          string
	mu          sync.Mutex
	state       vms.State
	properties  vms.Properties
	suspendable bool
	execCalls   []ExecCall

	// CloneBlock, if non-nil, causes Clone to block until the channel is
	// closed or the context is cancelled. Used by tests to pause a VM
	// mid-creation so the test can manipulate pool state before proceeding.
	CloneBlock chan struct{}

	CloneErr   error
	StartErr   error
	StopRunErr error
	StopErr    error
	StopState  *vms.State
	SuspendErr error
	DeleteErr  error
	ExecErr    error
}

// NewMock creates a new Mock VM instance.
func NewMock(id string) *Mock {
	return &Mock{
		state:       vms.StateInitial,
		suspendable: true,
		id:          id,
		properties:  vms.Properties{},
	}
}

func (m *Mock) ID() string {
	return m.id
}

func (m *Mock) Clone(ctx context.Context) error {
	if m.CloneBlock != nil {
		select {
		case <-m.CloneBlock:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.CloneErr != nil {
		return m.CloneErr
	}
	m.state = vms.StateStopped
	return nil
}

func (m *Mock) Start(ctx context.Context, _, _ io.Writer) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.StartErr != nil {
		m.state = vms.StateErrorUnknown
		return m.StartErr
	}
	m.state = vms.StateRunning
	go func() {
		<-ctx.Done()
		m.mu.Lock()
		if m.state == vms.StateRunning {
			m.state = vms.StateStopped
		}
		m.mu.Unlock()
	}()
	return nil
}

func (m *Mock) Stop(_ context.Context, _ time.Duration) (error, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == vms.StateStopped {
		return nil, nil
	}
	if m.state != vms.StateRunning {
		return nil, fmt.Errorf("cannot stop VM in state %v", m.state)
	}
	if m.StopRunErr != nil || m.StopErr != nil {
		m.state = vms.StateErrorUnknown
		return m.StopRunErr, m.StopErr
	}
	if m.StopState != nil {
		m.state = *m.StopState
	} else {
		m.state = vms.StateStopped
	}
	return nil, nil
}

func (m *Mock) Suspendable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.suspendable
}

func (m *Mock) SetSuspendable(suspendable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suspendable = suspendable
}

func (m *Mock) Suspend(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.SuspendErr != nil {
		m.state = vms.StateErrorUnknown
		return m.SuspendErr
	}
	m.state = vms.StateSuspended
	return nil
}

func (m *Mock) Delete(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state != vms.StateStopped && m.state != vms.StateSuspended && m.state != vms.StateErrorUnknown {
		return fmt.Errorf("cannot delete VM in state %v", m.state)
	}
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.state = vms.StateDeleted
	return nil
}

func (m *Mock) State(_ context.Context) vms.State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Mock) SetState(state vms.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

func (m *Mock) Exec(_ context.Context, _, _ io.Writer, cmd string, args ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state != vms.StateRunning {
		return fmt.Errorf("cannot exec command on VM in state %v", m.state)
	}
	m.execCalls = append(m.execCalls, ExecCall{Cmd: cmd, Args: slices.Clone(args)})
	if m.ExecErr != nil {
		return m.ExecErr
	}
	return nil
}

// ExecCalls returns all recorded Exec invocations.
func (m *Mock) ExecCalls() []ExecCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ExecCall(nil), m.execCalls...)
}

func (m *Mock) Properties(_ context.Context) (vms.Properties, error) {
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

// MockFactory creates and tracks Mock instances for pool and integration tests.
// Use Inject to pre-supply configured mocks; otherwise MockFactory.New creates
// plain NewMock instances on demand.
type MockFactory struct {
	suspendable bool
	mu          sync.Mutex
	mocks       []*Mock
	pending     []*Mock // pre-configured mocks to hand out first
}

// NewMockFactory returns an empty MockFactory.
func NewMockFactory(suspendable bool) *MockFactory {
	return &MockFactory{suspendable: suspendable}
}

var mockIDCounter atomic.Int32

// Inject queues m to be returned by the next New call instead of
// a freshly allocated Mock. Useful for injecting pre-configured error states.
func (f *MockFactory) Inject(m *Mock) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending = append(f.pending, m)
}

func (f *MockFactory) New() vms.Instance {
	f.mu.Lock()
	defer f.mu.Unlock()
	var m *Mock
	if len(f.pending) > 0 {
		m, f.pending = f.pending[0], f.pending[1:]
	} else {
		m = NewMock(fmt.Sprintf("mock-%d", mockIDCounter.Add(1)))
	}
	m.SetSuspendable(f.suspendable)
	f.mocks = append(f.mocks, m)
	return m
}

// Mocks returns a snapshot of all Mock instances produced so far.
func (f *MockFactory) Mocks() []*Mock {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*Mock(nil), f.mocks...)
}
