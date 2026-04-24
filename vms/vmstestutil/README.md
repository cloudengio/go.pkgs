# Package [cloudeng.io/vms/vmstestutil](https://pkg.go.dev/cloudeng.io/vms/vmstestutil?tab=doc)

```go
import cloudeng.io/vms/vmstestutil
```


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







