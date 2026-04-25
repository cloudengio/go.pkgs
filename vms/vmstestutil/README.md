# Package [cloudeng.io/vms/vmstestutil](https://pkg.go.dev/cloudeng.io/vms/vmstestutil?tab=doc)

```go
import cloudeng.io/vms/vmstestutil
```


## Functions
### Func RunPoolTests
```go
func RunPoolTests(t TestingT, cfg PoolTestConfig)
```
RunPoolTests runs a suite of integration tests that verify a
vmspool.Constructor creates instances that work correctly with vmspool.Pool.
Each scenario runs as a t.Run subtest so results appear individually and
failures are isolated.

Callers must provide a Constructor whose instances are real (or sufficiently
realistic) implementations of vms.Instance. The tests exercise the full
acquire → exec → release → replenish lifecycle.



## Types
### Type ExecCall
```go
type ExecCall struct {
	Cmd  string
	Args []string
}
```
ExecCall records a single invocation of Mock.Exec.


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
func NewMock() *Mock
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
func (m *Mock) Start(_ context.Context, _, _ io.Writer) error
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
func NewMockFactory() *MockFactory
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
	ExecCmd  string
	ExecArgs []string

	// Timeout caps individual pool operations. Defaults to 30 s.
	Timeout time.Duration

	// SupportsSuspend enables the suspend-mode subtests (WithSuspendVMs(true)).
	// Set this when the constructor produces instances that support Suspend.
	SupportsSuspend bool
}
```
PoolTestConfig configures the pool integration test suite run by
RunPoolTests.


### Type TestingT
```go
type TestingT interface {
	Helper()
	Run(name string, f func(*testing.T)) bool
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
	Cleanup(f func())
}
```
TestingT is the subset of *testing.T used by RunPoolTests. *testing.T
satisfies this interface directly.





