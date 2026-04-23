# Package [cloudeng.io/vms/vmspool](https://pkg.go.dev/cloudeng.io/vms/vmspool?tab=doc)

```go
import cloudeng.io/vms/vmspool
```

Package vmspool manages a fixed-size pool of suspended virtual machine
instances. The pool pre-creates and suspends VMs so they can be started
quickly when acquired. When a caller releases a VM it is deleted and a new
one is created asynchronously to restore the pool to its target size.

## Constants
### DefaultPoolSize
```go
DefaultPoolSize = 2

```



## Types
### Type Constructor
```go
type Constructor interface {
	New() vms.Instance
	Name() string
}
```
Constructor is a function that creates a new, uninitialized VM instance.
Each call must return a distinct instance.


### Type Option
```go
type Option func(*options)
```

### Functions

```go
func WithLogger(logger *slog.Logger) Option
```
WithLogger sets the logger used to report pool events and errors. The
default is the logger from the context at the time of Pool creation.


```go
func WithSize(size int) Option
```
WithSize sets the number of VMs to maintain in the pool. The default is
DefaultPoolSize.




### Type Pool
```go
type Pool struct {
	// contains filtered or unexported fields
}
```
Pool manages a fixed-size set of suspended virtual machine instances.

### Functions

```go
func New(ctx context.Context, constructor Constructor, opts ...Option) *Pool
```
New returns a Pool that will maintain size suspended VMs using constructor.
Call Start to fill the pool before calling Acquire.



### Methods

```go
func (p *Pool) Acquire(ctx context.Context) (*VM, error)
```
Acquire waits for a suspended VM, starts it, and returns a handle. The
caller must call VM.Release when finished with the VM. Acquire blocks until
a VM is available, ctx is cancelled, or the pool is closed.


```go
func (p *Pool) Close(ctx context.Context) error
```
Close stops accepting new acquires, waits for all replenishment goroutines
to finish, then deletes every suspended VM remaining in the pool.


```go
func (p *Pool) Start(ctx context.Context) error
```
Start fills the pool with size suspended VMs. It blocks until all VMs are
ready or any creation step fails. The context governs both the initial fill
and background replenishment goroutines launched during the pool's lifetime.




### Type VM
```go
type VM struct {
	// contains filtered or unexported fields
}
```
VM is a running virtual machine instance acquired from a Pool. Use Exec to
run commands and Release when done.

### Methods

```go
func (v *VM) Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error
```
Exec runs cmd with args inside the VM, writing output to stdout and stderr.


```go
func (v *VM) Release(ctx context.Context) error
```
Release deletes the VM and asynchronously replenishes the pool with a new
suspended instance. It must be called exactly once per acquired VM.







