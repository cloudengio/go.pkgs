# Package [cloudeng.io/vms/vmstestutil](https://pkg.go.dev/cloudeng.io/vms/vmstestutil?tab=doc)

```go
import cloudeng.io/vms/vmstestutil
```


## Functions
### Func TestInstanceCloneStartStopDelete
```go
func TestInstanceCloneStartStopDelete(t TestingT, cfg InstanceTestConfig)
```
TestInstanceCloneStartStopDelete verifies the standard Clone -> Start ->
Stop -> Delete state transitions.

### Func TestInstanceDeleteFromSuspended
```go
func TestInstanceDeleteFromSuspended(t TestingT, cfg InstanceTestConfig)
```
TestInstanceDeleteFromSuspended verifies that an instance can be deleted
directly from the Suspended state.

### Func TestInstanceExec
```go
func TestInstanceExec(t TestingT, cfg InstanceTestConfig)
```
TestInstanceExec verifies that a command can be executed inside a running
VM.

### Func TestInstanceLifecycle
```go
func TestInstanceLifecycle(t TestingT, cfg InstanceTestConfig)
```
TestInstanceLifecycle is a detailed lifecycle test that walks a VM through
its state machine. Initial → Clone → Stopped → Start → Running → Stop →
Stopped → Stop (idempotent) → Start → Running → [Suspend → Suspended →
Suspend (idempotent) → Start → Running →] Stop → Stopped → Delete → Deleted

### Func TestInstanceStateErrors
```go
func TestInstanceStateErrors(t TestingT, cfg InstanceTestConfig)
```

### Func TestInstanceSuspendResume
```go
func TestInstanceSuspendResume(t TestingT, cfg InstanceTestConfig)
```
TestInstanceSuspendResume verifies the Suspend and Resume (Start)
transitions for suspendable VMs.

### Func TestPoolAcquireExecRelease
```go
func TestPoolAcquireExecRelease(t TestingT, cfg PoolTestConfig)
```
TestPoolAcquireExecRelease verifies the full acquire → exec → release →
replenish cycle: releasing a VM triggers replenishment so the pool can serve
another Acquire.

### Func TestPoolClose
```go
func TestPoolClose(t TestingT, cfg PoolTestConfig)
```
TestPoolClose verifies that Close prevents further Acquire calls.

### Func TestPoolConcurrentAcquire
```go
func TestPoolConcurrentAcquire(t TestingT, cfg PoolTestConfig)
```
TestPoolConcurrentAcquire verifies that poolSize goroutines can each acquire
a VM concurrently without error, and that the pool replenishes after all are
released.

### Func TestPoolContextCancellation
```go
func TestPoolContextCancellation(t TestingT, cfg PoolTestConfig)
```
TestPoolContextCancellation verifies that Acquire returns context.Canceled
when the pool is empty and the context is cancelled.



## Types
### Type ExecCall
```go
type ExecCall struct {
	Cmd  string
	Args []string
}
```
ExecCall records a single invocation of Mock.Exec.


### Type InstanceTestConfig
```go
type InstanceTestConfig struct {
	// Constructor creates a new uninitialized vms.Instance for each test.
	Constructor vmspool.Constructor

	// Timeout caps individual operations. Defaults to 30 s.
	Timeout time.Duration

	// ExecCmd is a command that should succeed inside a running VM. If empty,
	// the Exec subtest is skipped.
	ExecCmd    string
	ExecArgs   []string
	ExecStdout string // Expected output from the exec.
	ExecStderr string // Expected stderr output from the exec.

	// RequireUnderlyingState is an optional helper for tests that need to verify
	// the underlying state of the instance, e.g. by querying a cloud provider API.
	// The function is expected to wait for the instance to reach a stable state
	RequireUnderlyingState func(
		ctx context.Context, inst vms.Instance, msg string, final vms.State, intermediate ...vms.State) error
}
```
InstanceTestConfig configures the test suite for an implementation of
vms.Instance.


### Type Mock
```go
type Mock struct {

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
	// contains filtered or unexported fields
}
```
Mock represents a mock virtual machine instance for testing.

### Functions

```go
func NewMock(id string) *Mock
```
NewMock creates a new Mock VM instance.



### Methods

```go
func (m *Mock) Clone(ctx context.Context) error
```


```go
func (m *Mock) Delete(_ context.Context) error
```


```go
func (m *Mock) Exec(_ context.Context, _, _ io.Writer, cmd string, args ...string) error
```


```go
func (m *Mock) ExecCalls() []ExecCall
```
ExecCalls returns all recorded Exec invocations.


```go
func (m *Mock) ID() string
```


```go
func (m *Mock) Properties(_ context.Context) (vms.Properties, error)
```


```go
func (m *Mock) SetProperties(props vms.Properties)
```


```go
func (m *Mock) SetState(state vms.State)
```


```go
func (m *Mock) SetSuspendable(suspendable bool)
```


```go
func (m *Mock) Start(ctx context.Context, _, _ io.Writer) error
```


```go
func (m *Mock) State(_ context.Context) vms.State
```


```go
func (m *Mock) Stop(_ context.Context, _ time.Duration) (error, error)
```


```go
func (m *Mock) Suspend(_ context.Context) error
```


```go
func (m *Mock) Suspendable() bool
```




### Type MockFactory
```go
type MockFactory struct {
	// contains filtered or unexported fields
}
```
MockFactory creates and tracks Mock instances for pool and integration
tests. Use Inject to pre-supply configured mocks; otherwise MockFactory.New
creates plain NewMock instances on demand.

### Functions

```go
func NewMockFactory(suspendable bool) *MockFactory
```
NewMockFactory returns an empty MockFactory.



### Methods

```go
func (f *MockFactory) Inject(m *Mock)
```
Inject queues m to be returned by the next New call instead of a freshly
allocated Mock. Useful for injecting pre-configured error states.


```go
func (f *MockFactory) Mocks() []*Mock
```
Mocks returns a snapshot of all Mock instances produced so far.


```go
func (f *MockFactory) New() vms.Instance
```




### Type PoolTestConfig
```go
type PoolTestConfig struct {
	// Constructor creates new VM instances. Required.
	Constructor vmspool.Constructor

	// PoolSize is the default pool size used across all tests. Defaults to 2.
	// Some subtests intentionally use a size-1 pool for deterministic behavior.
	PoolSize int

	// ExecCmd is a command that should succeed inside an acquired VM. If empty
	// the Exec subtest is skipped.
	ExecCmd          string
	ExecArgs         []string
	ExecStdoutOutput string // Expected output from the exec.
	ExecStderrOutput string // Expected stderr output from the exec.

	StdoutRWC func(string) io.Writer // Optional factory
	StderrRWC func(string) io.Writer // Optional factory for stderr RWC used by Exec; defaults to bytes.Buffer-based implementation.

	// Timeout caps individual pool operations. Defaults to 30 s.
	Timeout time.Duration

	// StagingBehaviour determines the pool's staging behaviour. Defaults to
	// StagingBehaviourRunning.
	StagingBehaviour vmspool.StagingBehaviour
}
```
PoolTestConfig configures the pool integration test suite run by
RunPoolTests.


### Type TestingT
```go
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
	Cleanup(f func())
}
```
TestingT is the subset of *testing.T used by RunPoolTests. *testing.T does
not satisfy this interface directly because Run's callback takes TestingT
rather than *testing.T; callers should wrap *testing.T with a thin adapter
(see pooltests_test.go for an example).





