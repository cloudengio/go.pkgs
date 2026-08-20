# Package [cloudeng.io/vms/vmspool](https://pkg.go.dev/cloudeng.io/vms/vmspool?tab=doc)

```go
import cloudeng.io/vms/vmspool
```

Package vmspool manages a fixed-size pool of suspended or stopped virtual
machine instances. The pool pre-creates and mantains VMs according to the
requested StagingBehaviour so they can be started quickly when acquired.
An acquired VM is deleted by VM.Delete, and a new one is created
asynchronously to restore the pool to its target size. A caller that calls
VM.StopAndRelease first triggers that replacement at the point of the stop
rather than at the delete, since a stopped VM will not run again.

## Constants
### DefaultPoolSize, DefaultCleanupTimeout, DefaultStopTimeout
```go
DefaultPoolSize = 2
DefaultCleanupTimeout = time.Minute
DefaultStopTimeout = time.Minute

```



## Functions
### Func DefaultCreateBackoff
```go
func DefaultCreateBackoff() ratecontrol.ExponentialBackoffConfig
```
DefaultCreateBackoff returns the default backoff used when creating VMs: a
500ms initial delay doubling over 10 steps, for a total delay budget of ~8.5
minutes, which is also the timeout allowed for a single creation attempt.



## Types
### Type Config
```go
type Config struct {
	Size                  int                                  `yaml:"size" doc:"The number of VMs to maintain in the pool. A 0 or negative value is treated as DefaultPoolSize."`
	CleanupTimeout        time.Duration                        `yaml:"cleanup_timeout" doc:"The timeout for cleaning up VMs during Acquire and Close. A 0 or negative value is treated as DefaultCleanupTimeout."`
	CreateBackoff         ratecontrol.ExponentialBackoffConfig `yaml:"create_backoff" doc:"The backoff applied between attempts to create a VM. Its total delay budget also bounds a single creation attempt. A configuration with a non-positive initial delay or number of steps is treated as DefaultCreateBackoff."`
	StopTimeout           time.Duration                        `yaml:"stop_timeout" doc:"The timeout for stopping VMs. A 0 or negative value is treated as DefaultStopTimeout."`
	StagingBehaviour      StagingBehaviour                     `yaml:"staging_behaviour" doc:"The staging behaviour for VMs in the pool. The default is StagingBehaviourRunning. The behaviours are: StagingBehaviourRunning: VMs are left running and Acquire will hand them to the caller as-is. StagingBehaviourSuspended: VMs are suspended and Acquire will resume them before handing them to the caller provided that the VM supports suspend/resume; if not, the pool falls back to StagingBehaviourStopped behaviour. StagingBehaviourStopped: VMs are stopped and Acquire will start them before handing them to the caller."`
	DeleteAcquiredOnClose bool                                 `yaml:"delete_acquired_on_close" doc:"Controls whether Close deletes VMs that are still held by a caller, ie. that have been acquired but not yet deleted, whether or not the caller has stopped them with VM.StopAndRelease. The default is true, on the basis that the pool owns every VM it creates and Close is the last chance to delete them. Set it to false when callers outlive the pool and are responsible for calling VM.Delete themselves; Close then emits EventAcquiredVMRetained for each VM it leaves behind."`
}
```

### Methods

```go
func (c Config) Options() []Option
```
Options returns a slice of Option values derived from the Config fields.
Zero or negative durations and sizes are left to the individual With*
functions to replace with their documented defaults. It does not include
WithStatus or WithStdoutStderr, which require non-serialisable values
(channels, functions).




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
### EventAcquireWaiting, EventVMDequeued, EventAcquired, EventAcquireFailed, EventAttemptToUseClosedPool, EventRelease, EventReleased, EventVMCreateStarted, EventVMCreated, EventVMCreateFailed, EventReplenishStarted, EventReplenished, EventReplenishFailed, EventStartPoolFull, EventOrphanedVMDeleted, EventAcquiredVMRetained
```go
// EventAcquireWaiting is emitted when Acquire is called and blocks
// waiting for a suspended VM to become available.
EventAcquireWaiting EventKind = iota
// EventVMDequeued is emitted when a suspended or running VM is taken from
// the pool and is about to be started by the caller, or if running
// returned as-is to the caller.
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
// EventRelease is emitted when VM.Delete is called by the caller.
EventRelease
// EventReleased is emitted after the VM has been deleted and
// replenishment has been scheduled.
EventReleased
// EventVMCreateStarted is emitted when a goroutine is launched to create a new VM
// to place in the pool.
EventVMCreateStarted
// EventVMCreated is emitted when a new VM has been successfully created.
EventVMCreated
// EventVMCreateFailed is emitted when VM creation fails.
EventVMCreateFailed
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
// EventStartPoolFull is emitted when the asynchronous process to
// fill the pool initiated by Start is completed.
EventStartPoolFull
// EventOrphanedVMDeleted is emitted by Close for each VM it deletes that
// was not waiting in the pool: one abandoned part way through creation, or
// one still held by a caller that never deleted it.
EventOrphanedVMDeleted
// EventAcquiredVMRetained is emitted by Close for each acquired VM it
// leaves in place because WithDeleteAcquiredOnClose(false) was set. The
// caller that holds the VM is responsible for deleting it.
EventAcquiredVMRetained

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
func WithCleanupTimeout(timeout time.Duration) Option
```
WithCleanupTimeout sets the timeout for cleaning up VMs during Acquire
and Close. The default is DefaultCleanupTimeout. A 0 or negative value is
treated as DefaultCleanupTimeout.


```go
func WithCreateBackoff(cfg ratecontrol.ExponentialBackoffConfig) Option
```
WithCreateBackoff sets the backoff applied between attempts to create
a VM. Its total delay budget also bounds how long a single creation
attempt may take before it is abandoned and retried. The default is
DefaultCreateBackoff(); a configuration with a non-positive initial delay
or number of steps is replaced by it, so that an unset configuration cannot
produce a NoBackoff that retries without pausing.


```go
func WithDeleteAcquiredOnClose(v bool) Option
```
WithDeleteAcquiredOnClose controls whether Close deletes VMs that are
still held by a caller, ie. that have been acquired but not yet deleted,
whether or not the caller has stopped them with VM.StopAndRelease.
The default is true, on the basis that the pool owns every VM it creates
and Close is the last chance to delete them. Set it to false when callers
outlive the pool and are responsible for calling VM.Delete themselves;
Close then emits EventAcquiredVMRetained for each VM it leaves behind.


```go
func WithSize(size int) Option
```
WithSize sets the number of VMs to maintain in the pool. The default is
DefaultPoolSize. A 0 or negative value is treated as DefaultPoolSize.


```go
func WithStagingBehaviour(behaviour StagingBehaviour) Option
```
WithStagingBehaviour sets the staging behaviour for VMs in the pool.
The default is StagingBehaviourRunning.


```go
func WithStatus(ch chan<- Event) Option
```
WithStatus registers ch to receive pool lifecycle events. Sends are
non-blocking: events are dropped if ch is full. The caller is responsible
for sizing the channel appropriately and draining it promptly.


```go
func WithStdoutStderr(stdout, stderr func(id string) io.Writer) Option
```
WithStdoutStderr configures the pool to use the provided functions to create
stdout and stderr io.Writers for VMs during creation and replenishment.
The value of vms.Instance.ID() is passed to the stdout function and can
be used to create uniquely identifiable pipes. If either function is nil,
a no-op Writer is used that discards all writes.


```go
func WithStopTimeout(timeout time.Duration) Option
```
WithStopTimeout sets the timeout for stopping VMs. The default is
DefaultStopTimeout. A 0 or negative value is treated as DefaultStopTimeout.




### Type Pool
```go
type Pool struct {
	// contains filtered or unexported fields
}
```
Pool manages a fixed-size set of suspended virtual machine instances.

### Functions

```go
func New(provider Provider, opts ...Option) *Pool
```
New returns a Pool that will maintain size suspended VMs using provider.
Call Start to fill the pool before calling Acquire.



### Methods

```go
func (p *Pool) Acquire(ctx context.Context) (*VM, error)
```
Acquire waits for a suspended VM, starts it, and returns a handle. The
caller must call VM.Delete when finished with the VM. Acquire blocks until a
VM is available, ctx is cancelled, or the pool is closed.


```go
func (p *Pool) Close(ctx context.Context) error
```
Close stops accepting new acquires, waits for all replenishment goroutines
to finish, then deletes every VM the pool created whose deletion has not
already been performed, or claimed, by another path (such as a concurrent
Delete). That includes VMs queued in the pool, VMs abandoned part way
through creation, and, unless WithDeleteAcquiredOnClose(false) was used,
VMs that a caller acquired and has not deleted. Close is idempotent.

A cancelled ctx bounds only the wait for the in-flight creation goroutines:
Close then attempts to delete their VMs while the creation operations are
still running. Such attempts fail with 'unexpected VM state' errors that
are included in the returned error, and those VMs are instead deleted by the
creation goroutines' own cleanup once they observe the cancellation or the
closed pool. Deletion itself is not bounded by ctx.


```go
func (p *Pool) Start(ctx context.Context) error
```
Start blocks until at least one VM is ready to be acquired (or the
context is canceled), any other VMs required to fill the pool are created
asynchronously. Start can be called once only and will return an error if
called more than once. After Start returns, the pool is ready to accept
Acquire calls.




### Type Provider
```go
type Provider interface {
	// New returns a new, uninitialized VM instance. Each call must return a
	// distinct vms.Instance. ctx governs any work done to construct the
	// instance. It returns an error if the instance could not be created.
	New(ctx context.Context) (vms.Instance, error)
	// List returns lightweight summaries of the VMs currently present for this
	// provider's pool.
	List(ctx context.Context) ([]VMInfo, error)
	// Get returns the full details of a single VM by name.
	Get(ctx context.Context, vmName string) (VMDetail, error)
	// Delete stops (if running) and deletes every VM belonging to this
	// provider's pool, returning the names deleted. It continues past individual
	// failures.
	Delete(ctx context.Context, stopTimeout time.Duration) ([]string, error)
}
```
Provider creates and manages the VMs for a Pool. In addition to constructing
new instances it can enumerate, inspect and delete the VMs it has created,
which pools and cleanup tooling use for status reporting and reclaiming
orphaned VMs.


### Type StagingBehaviour
```go
type StagingBehaviour int
```
StagingBehaviour determines the state of VMs in the pool after creation but
before acquisition. The default is StagingBehaviourRunning. The behaviours
are:
  - StagingBehaviourRunning: VMs are left running and Acquire will hand them
    to the caller as-is.
  - StagingBehaviourSuspended: VMs are suspended and Acquire will resume
    them before handing them to the caller provided that the VM supports
    suspend/resume; if not, the pool falls back to StagingBehaviourStopped
    behaviour.
  - StagingBehaviourStopped: VMs are stopped and Acquire will start them
    before handing them to the caller.

### Constants
### StagingBehaviourRunning, StagingBehaviourSuspended, StagingBehaviourStopped
```go
StagingBehaviourRunning StagingBehaviour = iota
StagingBehaviourSuspended
StagingBehaviourStopped

```



### Methods

```go
func (s StagingBehaviour) MarshalText() ([]byte, error)
```
MarshalText implements encoding.TextMarshaler, emitting the string name of
the behaviour. yaml.v3, encoding/json, and other text-based encoders will
call this automatically.


```go
func (s StagingBehaviour) String() string
```


```go
func (s *StagingBehaviour) UnmarshalText(b []byte) error
```
UnmarshalText implements encoding.TextUnmarshaler, accepting the string name
of the behaviour case-insensitively ("Running", "Suspended", "Stopped").
yaml.v3 calls this for string-valued YAML nodes, so no direct yaml import is
needed in this package.




### Type VM
```go
type VM struct {
	// contains filtered or unexported fields
}
```
VM is a running virtual machine instance acquired from a Pool. Use Exec to
run commands and Delete when done.

### Methods

```go
func (v *VM) Delete(ctx context.Context) error
```
Delete deletes the VM, stopping it first if it is still running. It must be
called exactly once per acquired VM. If the pool has been closed and deletes
acquired VMs on close (the default), the VM has already been deleted by
Close and Delete does not delete it again.

Delete asynchronously replenishes the pool unless the VM has already been
stopped by StopAndRelease, which released the slot at that point; requesting
a second replacement would grow the pool beyond its configured size.


```go
func (v *VM) Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error
```
Exec runs cmd with args inside the VM, writing output to stdout and stderr.


```go
func (v *VM) ID() string
```
ID returns the unique identifier of the VM instance. It may be empty if the
VM is invalid.


```go
func (v *VM) StopAndRelease(ctx context.Context, timeout time.Duration) (runErr, stopErr error)
```
StopAndRelease stops the VM and releases its slot in the pool, returning any
error from the last command run in the VM and any error from stopping it.
It is idempotent.

A stopped VM is never handed out again, so the first successful stop
asynchronously replenishes the pool rather than leaving the slot idle:
the replacement is created while the caller is still finishing with the
stopped VM (collecting logs, say). The VM itself is not deleted; the caller
must still call Delete when done with it, which will not request a second
replacement.




### Type VMDetail
```go
type VMDetail struct {
	VMInfo
	DiskGiB  int // size of the VM's disk in GiB
	NumCores int // number of CPU cores allocated to the VM
	MemGiB   int // amount of RAM allocated to the VM in GiB
}
```
VMDetail extends VMInfo with the fuller, potentially more expensive per-VM
details returned by Provider.Get, such as the resources allocated to the VM.


### Type VMInfo
```go
type VMInfo struct {
	Name     string
	Pool     string
	State    string // backend-specific state string, e.g. "running", "stopped"
	Running  bool
	Accessed time.Time // best-effort last activity time; may be creation time if last access is unavailable
}
```
VMInfo is a backend-neutral summary of a VM managed by a Provider.
It holds only the lightweight fields that are cheap to obtain in bulk via
Provider.List.





