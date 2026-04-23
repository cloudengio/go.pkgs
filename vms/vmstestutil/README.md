# Package [cloudeng.io/vms/vmstestutil](https://pkg.go.dev/cloudeng.io/vms/vmstestutil?tab=doc)

```go
import cloudeng.io/vms/vmstestutil
```


## Types
### Type Mock
```go
type Mock struct {
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
func (m *Mock) Clone(_ context.Context) error
```


```go
func (m *Mock) Delete(_ context.Context) error
```


```go
func (m *Mock) Exec(_ context.Context, _, _ io.Writer, _ string, _ ...string) error
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







