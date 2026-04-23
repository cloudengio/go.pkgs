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


### Type Event
```go
type Event struct {
	Time time.Time
	Kind EventKind
	Err  error // non-nil for *Failed events
}
```
Event describes a single pool lifecycle event.


### Type EventKind
```go
type EventKind int
```
EventKind identifies the type of pool event sent to a status channel.

### Constants
### EventAcquireWaiting, EventVMDequeued, EventAcquired, EventAcquireFailed, EventAttemptToUseClosedPool, EventRelease, EventReleased, EventReplenishStarted, EventReplenished, EventReplenishFailed
```go
// EventAcquireWaiting is emitted when Acquire is called and blocks
// waiting for a suspended VM to become available.
EventAcquireWaiting EventKind = iota
// EventVMDequeued is emitted when a suspended VM is taken from the pool
// and is about to be started for the caller.
EventVMDequeued
// EventAcquired is emitted when the VM has been started and is returned
// to the caller.
EventAcquired
// EventAcquireFailed is emitted when Acquire returns an error (context
// cancelled or VM start failure). Err is set.
EventAcquireFailed
// EventAttemptToUseClosedPool is emitted when Acquire is called on a pool
// that is already closed or has been signalled to close. Err is set.
EventAttemptToUseClosedPool
// EventRelease is emitted when Release is called by the caller.
EventRelease
// EventReleased is emitted after the VM has been deleted and
// replenishment has been scheduled.
EventReleased
// EventReplenishStarted is emitted when a replenishment goroutine is
// launched to restore the pool to its target size.
EventReplenishStarted
// EventReplenished is emitted when a new VM has been suspended and
// placed in the pool, restoring one unit of capacity.
EventReplenished
// EventReplenishFailed is emitted when VM creation during replenishment
// fails. The pool shrinks by one until a later replenishment succeeds.
// Err is set.
EventReplenishFailed

```



### Methods

```go
func (e EventKind) String() string
```




### Type Option
```go
type Option func(*options)
```

### Functions

```go
func WithSize(size int) Option
```
WithSize sets the number of VMs to maintain in the pool. The default is
DefaultPoolSize. A 0 or negative value is treated as DefaultPoolSize.


```go
func WithStatus(ch chan<- Event) Option
```
WithStatus registers ch to receive pool lifecycle events. Sends are
non-blocking: events are dropped if ch is full. The caller is responsible
for sizing the channel appropriately and draining it promptly.




### Type Pool
```go
type Pool struct {
	// contains filtered or unexported fields
}
```
Pool manages a fixed-size set of suspended virtual machine instances.

### Functions

```go
func New(constructor Constructor, opts ...Option) *Pool
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







