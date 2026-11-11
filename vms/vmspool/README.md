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
const (
	DefaultPoolSize       = 2
	DefaultCleanupTimeout = time.Minute
	DefaultStopTimeout    = time.Minute
)
```

### DefaultStatusBuffer
```go
const DefaultStatusBuffer = 100
```
DefaultStatusBuffer is the number of events a StatusTracker buffers for a
pool whilst they are waiting to be counted, see NewStatusTracker.



## Variables
### ErrNoSuchPool, ErrPoolsClosed
```go
// ErrNoSuchPool is returned for a name the set holds no pool for.
ErrNoSuchPool = errors.New("no such pool")
// ErrPoolsClosed is returned by the operations of a closed set.
ErrPoolsClosed = errors.New("pools is closed")

```

### ErrNoProvider
```go
ErrNoProvider = errors.New("no VM provider configured")

```
ErrNoProvider should be returned by any operation that requires a VM
provider when no appropriate provider has been configured.



## Functions
### Func DefaultCreateBackoff
```go
func DefaultCreateBackoff() ratecontrol.ExponentialBackoffConfig
```
DefaultCreateBackoff returns the default backoff used when creating VMs: a
500ms initial delay doubling over 10 steps, for a total delay budget of ~8.5
minutes, which is also the timeout allowed for a single creation attempt.

### Func DeleteVMs
```go
func DeleteVMs(ctx context.Context, stopTimeout time.Duration, providerNew ...ProviderNew) ([]string, error)
```
DeleteVMs deletes all VMs from the given providers, sorted by VM name.
It returns a slice of deleted VM IDs and any error encountered. Errors are
accumulated from all providers. Note, that this function is intentionally
independent of any pool and operates directly on the providers.



## Types
### Type CachingProvider
```go
type CachingProvider struct {
	Provider
	// contains filtered or unexported fields
}
```
CachingProvider wraps a Provider and caches the per-VM details that are
expensive to obtain, which for a provider backed by a command line tool
means one invocation of it per VM. What it caches is the part of a VMDetail
that does not change whilst a VM exists: the resources it was created with.
The state of a VM does change, so GetAll never serves that from the cache,
see the VMInfo rule there.

A zero CachingProvider wrapping a Provider is ready to use, and it is safe
for concurrent use. It is a Provider itself, so a pool can be created with
one, and it caches nothing a pool itself asks for: a pool creates VMs and
leaves reporting on them to its application.

### Methods

```go
func (c *CachingProvider) Get(ctx context.Context, vmName string) (VMDetail, error)
```
Get implements Provider, returning the full details of a single VM from the
cache if they are held there and reading them from the wrapped Provider if
they are not. Missing resources are filled in from the provider's defaults,
see Provider.DefaultResources, and nothing is cached for a VM that cannot be
read.

A cached VM is served entirely from the cache, its VMInfo included, which
may be as old as the entry: use GetAll where the state has to be current.
Nothing is dropped from the cache here either, since one VM says nothing
about which others still exist; it is GetAll that prunes it.


```go
func (c *CachingProvider) GetAll(ctx context.Context) ([]VMDetail, []string, error)
```
GetAll returns the full details of every VM the wrapped Provider lists,
reading the details of each VM it has not seen before and serving the rest
from the cache. VMs the listing no longer reports are dropped from it, so it
holds the VMs that exist rather than every VM the provider has ever had.

The listing is fresher than anything cached, so each VMDetail carries the
VMInfo the listing gave it: a cached VM reports the state it is in now,
not the state it was in when its details were read.

A VM whose details cannot be read is still returned, carrying what the
listing said about it and the provider's default resources: a VM that exists
and cannot be inspected is of more use reported than omitted, not least
because the count of VMs would otherwise disagree with the pool's own.
Its name is returned in failed, and nothing is cached for it, so the next
call reads it again rather than leaving it reporting defaults for as long as
it exists. The error accumulates those failures and is returned alongside
results that are usable whether or not it is nil; only a listing that fails
returns no results at all.

The details are returned ordered by VM name, whatever order List reports.




### Type CompletedVM
```go
type CompletedVM interface {
	// GetVM returns the VM the work ran on, or nil if it had none, in which
	// case there is nothing for the queue to release.
	GetVM() *VM
	// GetLogger returns the logger to report this event with. The logger
	// is likely preconfigured with per-vm, per-job fields.
	GetLogger(*slog.Logger) *slog.Logger
}
```
CompletedVM represents the operations available on a completed VM.


### Type CompletionEvent
```go
type CompletionEvent struct {
	VM      CompletedVM
	Err     error  // The error that occurred during the execution of the VM, if any.
	Outcome string // The outcome of the work performed by the VM, used to categorize the completion event.
	// contains filtered or unexported fields
}
```
CompletionEvent represents a single occurrence of a completed VM any error
that occurred related to the execution of the VM, as opposed to the outcome
of the work the VM was performing which is captured in the Outcome field.


### Type CompletionOutcomeOption
```go
type CompletionOutcomeOption func(*CompletionQueue)
```
CompletionOutcomeOption represents an option for configuring a completion
outcome in the queue.

### Functions

```go
func WithCompletionOutcome(name string, capacity int, expiration time.Duration) CompletionOutcomeOption
```
WithCompletionOutcome configures a completion outcome in the queue with the
given name, capacity, and expiration duration.




### Type CompletionQueue
```go
type CompletionQueue struct {
	// contains filtered or unexported fields
}
```
CompletionQueue parks finished work, and the VM that ran it, for a period of
time before the VM is deleted, so that it can be inspected after the fact.
Work is queued by outcome, each outcome configured with its own capacity and
retention by WithCompletionOutcome, so that a failure can be kept for longer
than a success, which is usually what is wanted.

A single completion queue may be shared across multiple VM pools; it reports
the VMs it is holding both in total and per pool, so that each pool can
account for the VMs it created that the queue has yet to release.

### Functions

```go
func NewCompletionQueue(ctx context.Context, opts ...CompletionOutcomeOption) *CompletionQueue
```
NewCompletionQueue returns a queue with one sub-queue per outcome configured
by WithCompletionOutcome, each holding at most that outcome's capacity of
events and retaining each for that outcome's expiration before deleting the
VM it was holding. An event whose Outcome names no configured sub-queue is
discarded by Push, so a queue created with no options holds nothing.

The queue's goroutines are governed by ctx, so cancelling it ends them;
Close copes with a queue whose context has already been cancelled, which is
what happens on an interrupt.



### Methods

```go
func (q *CompletionQueue) Close(ctx context.Context) error
```
Close drains every outcome's queue and deletes the VMs they were holding.
It is idempotent, since a queue shared by several pools is closed by each of
them: a later call returns the result of the first.


```go
func (q *CompletionQueue) NumEvents() int
```
NumEvents returns the number of VMs the queue is holding across all of its
outcomes. An event that was pushed without a VM is not counted: there is
nothing for the queue to release and nothing for a caller to report.


```go
func (q *CompletionQueue) NumEventsPerPool(poolName string) int
```
NumEventsPerPool returns the number of VMs the queue is holding that came
from the named pool, counted as NumEvents counts them.


```go
func (q *CompletionQueue) Out(outcome string) <-chan CompletionEvent
```
Out returns the channel of events for the requested outcome, or nil if no
such outcome was configured or the queue is closed.


```go
func (q *CompletionQueue) Push(event CompletionEvent)
```
Push adds a completion event to the queue for its specified outcome.
An event whose Outcome names no configured outcome is discarded, as is
one pushed to a closed queue or to one whose context has been cancelled:
in each case nothing would ever release the VM, which is left to the pool
that created it to delete.


```go
func (q *CompletionQueue) Remove(poolName, vmID string)
```
Remove stops tracking the VM, for a caller that has taken responsibility for
it, such as one draining the queue. Removing a VM the queue is not holding
does nothing.


```go
func (q *CompletionQueue) Subscribe(ctx context.Context) (<-chan struct{}, func())
```
Subscribe returns a channel that is signalled whenever the set of VMs
in the queue changes, and a function that must be called to release the
subscription.


```go
func (q *CompletionQueue) VMIDs() []string
```
VMIDs returns a copy of the set of VM IDs currently parked in the queue,
in no particular order.


```go
func (q *CompletionQueue) VMIDsForPool(pool string) []string
```
VMIDsForPool returns a copy of the set of VM IDs currently parked in the
queue for the specified pool.




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
	// Pool is the name of the pool that emitted the event, see Pool.Name. It
	// is always set, so that a consumer of a channel shared by several pools
	// can attribute what it receives without a channel per pool.
	Pool string
	// VM is the ID of the VM the event concerns, empty for an event about the
	// pool itself rather than one of its VMs. It is also empty for the events
	// of a VM that does not exist yet or no longer exists: creation that
	// failed before there was anything to name, and the restoration of a
	// pool's capacity, which EventVMCreated has already named the new VM for.
	VM  string
	Err error // non-nil for *Failed events
}
```
Event describes a single pool lifecycle event.


### Type EventKind
```go
type EventKind int
```
EventKind identifies the type of pool event sent to a status channel.

### Constants
### EventAcquireWaiting, EventVMDequeued, EventAcquired, EventAcquireFailed, EventAttemptToUseClosedPool, EventRelease, EventReleased, EventVMCreateStarted, EventVMCreated, EventVMCreateFailed, EventReplenishStarted, EventReplenished, EventReplenishFailed, EventStartPoolFull, EventOrphanedVMDeleted, EventAcquiredVMRetained, EventRefreshFailed
```go
const (
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

	// EventRefreshFailed is emitted by Refresh for each VM it took out of the
	// pool but could not delete. VM is set, and Err, which names it too so
	// that the error is self-contained wherever it is reported. The pool is
	// replenished for that VM regardless, so it keeps its size, but the VM
	// itself may be left behind for an external sweep to find.
	EventRefreshFailed
)
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
func WithCompletionQueue(q *CompletionQueue) Option
```
WithCompletionQueue attaches a completion queue to the pool, which the
VMs it hands out report through VM.CompletionQueue. It is how work
running on a VM finds the queue to park that VM in when it is done,
without the queue being plumbed through alongside the VM, and how an
application reports on the VMs of this pool that the queue is still holding,
see CompletionQueue.NumEventsPerPool.

A single queue may be attached to any number of pools. The pool neither
pushes to the queue nor closes it: parking finished work is the caller's
decision, and the queue, which outlives any one pool that shares it,
is closed by whoever created it.


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
func New(provider Provider, name string, opts ...Option) *Pool
```
New returns a Pool, named name, that will maintain size suspended VMs using
provider. The name identifies the pool to an application that runs more than
one and is reported by the VMs it hands out, see VM.PoolName; it is not used
to name the VMs themselves, which the provider names. Call Start to fill the
pool before calling Acquire.



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
func (p *Pool) CompletionQueue() *CompletionQueue
```
CompletionQueue returns the queue that finished work from this pool is
parked in, or nil if the pool has none, see WithCompletionQueue. All VMs
from this pool return this value via their CompletionQueue method.


```go
func (p *Pool) Name() string
```
Name returns the name the pool was created with. All VMS from this pool
return this value via their PoolName method.


```go
func (p *Pool) Provider() Provider
```
Provider returns the provider the pool creates its VMs with, see New.
It is fixed when the pool is created, and is exposed so that an application
wrapping a Pool can reach the provider it supplied - to report what the VMs
were created from, or to list and inspect them - without having to keep a
second reference to it alongside the pool.


```go
func (p *Pool) Refresh(ctx context.Context) int
```
Refresh takes the VMs that are currently available out of the pool and
deletes them, leaving the pool to create replacements, which is how a pool
picks up a new base image without being restarted. It returns the number of
VMs removed.

VMs that are held by a caller are left alone: they are in use, and the work
using them did not ask for a different image. Nor does Refresh wait for
the replacements, which take as long as creating any VM does, or for the
deletions themselves: it returns as soon as the VMs it is replacing can no
longer be handed out. A VM that cannot be deleted is reported through the
status channel as EventRefreshFailed.

Refresh returns 0 for a pool with nothing available to replace, including a
closed one.


```go
func (p *Pool) Size() int
```
Size returns the number of VMs the pool is configured to hold, see WithSize.
It is fixed when the pool is created, so it is not part of Stats.


```go
func (p *Pool) Start(ctx context.Context) error
```
Start blocks until at least one VM is ready to be acquired (or the
context is canceled), any other VMs required to fill the pool are created
asynchronously. Start can be called once only and will return an error if
called more than once. After Start returns, the pool is ready to accept
Acquire calls.


```go
func (p *Pool) Stats() Stats
```
Stats returns a snapshot of the pool's size and how many of its VMs are
available, acquired and pending. It does not wait for any operation in
progress, such as a VM being started by Acquire, and is safe to call from
any goroutine at any time, including before Start and after Close.

The counts are read together but the pool keeps changing while they are,
so they describe a moment that may already have passed; a VM being handed
over by Acquire, say, is counted as Pending until the caller holds it.
The counts are consistent with each other, however: none is ever negative,
and Available + Acquired + Pending is the number of VMs the pool has created
and not yet deleted, which can exceed Size while replacements for stopped
VMs are created.




### Type Provider
```go
type Provider interface {
	// Name returns the name of the provider, typically the infrastructure
	// used to implement the pool such as 'tart' or 'docker'. This will be
	// the same as ProviderNew.Name() if an instance of ProviderNew was used
	// to create this provider.
	Name() string

	// DefaultResources returns the default resource allocation for new VMs
	// created by this provider, these may be overridden when creating individual
	// VMs.
	DefaultResources() VMResources

	// Details returns provider-specific details about the infrastructure.
	Details() ProviderDetails

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


### Type ProviderDetails
```go
type ProviderDetails interface {
	// Image returns the reference the provider's VMs are created from, or the
	// zero Reference if the provider has no single image or does not know it,
	// see ociimage.Reference.IsZero. It is parsed rather than a string so that
	// a caller naming something after the image, or deciding whether it can be
	// pulled, does not have to take the reference apart again.
	Image() ociimage.Reference
}
```
ProviderDetails defines an interface for provider-specific details about the
infrastructure implementing the VM provider.


### Type ProviderNew
```go
type ProviderNew interface {
	// Name is the name of the VM technology used and will be the value of
	// Name returned by the created provider.
	Name() string
	// New builds a provider from the underlying state or configuration.
	New(context.Context) (Provider, error)
}
```
ProviderNew is used to construct a provider from some underlying state or
configuration.


### Type Set
```go
type Set struct {
	// contains filtered or unexported fields
}
```
Set is a set of named pools that are held and closed together, along with
the completion queue and the status tracker they share.

### Functions

```go
func NewSet(cq *CompletionQueue, tracker *StatusTracker, pools ...*Pool) *Set
```
NewSet returns a set holding pools, keyed by the name each was created with,
and already started, see Pool.Start. cq and tracker are what the pools share
and either may be nil; the set owns both and closes them after the pools,
see Close. Two pools of one name are one pool, the last winning.



### Methods

```go
func (s *Set) Acquire(ctx context.Context, poolName string) (*VM, error)
```
Acquire returns a VM from the named pool, see Pool.Acquire.


```go
func (s *Set) All() map[string]*Pool
```
All returns every pool, keyed by name, as a map of the caller's own.
A closed set returns its pools too: a closed pool still reports what it is.


```go
func (s *Set) AvailableAndPending() (available, pending bool)
```
AvailableAndPending reports whether any pool has a VM that Acquire can
return without waiting for one to be created, and whether any pool has
one being created and so acquirable shortly, see Pool.Stats. The two are
independent: a set filling itself is both. A closed set is neither.


```go
func (s *Set) Close(ctx context.Context) error
```
Close closes every pool concurrently, then the completion queue and the
status tracker, accumulating their errors. The set is out of service from
the moment it is called.

The queue goes after the pools, since closing it deletes the VMs it holds,
and the tracker last, since a pool reports to it until it has stopped.
It is idempotent: a later call returns the first one's result.


```go
func (s *Set) CompletionQueue() *CompletionQueue
```
CompletionQueue returns the queue the pools' finished VMs are parked in,
or nil if the set has none. The set owns it: fill and drain it, do not close
it.


```go
func (s *Set) Len() int
```
Len returns the number of pools in the set.


```go
func (s *Set) Names() []string
```
Names returns the names of the pools, sorted, so that listings are stable.


```go
func (s *Set) Pool(name string) (*Pool, error)
```
Pool returns the named pool, ErrNoSuchPool if the set holds none of that
name, or ErrPoolsClosed if the set has been closed.


```go
func (s *Set) Refresh(ctx context.Context, poolName string) (int, error)
```
Refresh replaces the VMs the named pool has available and returns how many
were replaced, see Pool.Refresh.


```go
func (s *Set) StatusTracker() *StatusTracker
```
StatusTracker returns the tracker counting the pools' events, or nil if the
set has none. The set owns it.


```go
func (s *Set) Subscribe(ctx context.Context) (<-chan struct{}, func())
```
Subscribe returns a channel signalled whenever any pool's counters
change and a function that must be called to release the subscription,
see StatusTracker.Subscribe.

A set with no tracker has no signal: the channel is nil, which blocks
forever rather than reporting changes that are not being counted, and
releasing it does nothing.




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
const (
	StagingBehaviourRunning StagingBehaviour = iota
	StagingBehaviourSuspended
	StagingBehaviourStopped
)
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




### Type Stats
```go
type Stats struct {
	// Available is the number of VMs waiting in the pool that Acquire can
	// return without waiting for one to be created.
	Available int
	// Acquired is the number of VMs held by callers that have not yet deleted
	// them. A VM stopped by VM.StopAndRelease is still counted, since the
	// caller has yet to call VM.Delete, even though its slot in the pool has
	// already been given up and is being replenished.
	Acquired int
	// Pending is the number of VMs that exist but are neither available nor
	// acquired: those being created, whether by Start or to replenish the
	// pool, and those in transit between the pool and a caller.
	Pending int
}
```
Stats is a snapshot of the state of a Pool, see Pool.Stats.

### Methods

```go
func (s Stats) String() string
```
String implements fmt.Stringer.




### Type StatusTracker
```go
type StatusTracker struct {
	// contains filtered or unexported fields
}
```
StatusTracker counts the lifecycle events of one or more pools by kind,
logs them, and notifies its subscribers whenever those counts change.
It is safe for concurrent use.

It uses nothing of a pool but its public API: a pool reports to a channel,
see WithStatus, and the tracker counts what arrives there. Every event names
the pool it came from, see Event.Pool, so one tracker serves a whole set of
pools and its change signal covers all of them.

### Functions

```go
func NewStatusTracker(ctx context.Context, buffer int) *StatusTracker
```
NewStatusTracker returns a tracker and starts the goroutine that counts
and logs what the pools report, which Close ends. buffer is how many of a
pool's events may be waiting to be counted before the oldest are dropped,
and is the capacity of each channel Track creates; 0 or less is
DefaultStatusBuffer.

Events are logged as they are counted, through the logger carried by ctx
at Info, or Error for an event carrying one: that is the only report a
pool's asynchronous failures ever get. ctx is used for its logger alone,
so cancelling it does not stop the counting; Close does.



### Methods

```go
func (t *StatusTracker) Close()
```
Close ends the counting, every subscription and the forwarding of every
channel the tracker was given; later events are discarded. It is idempotent,
and the counters it accumulated stand.


```go
func (t *StatusTracker) Counters(pool string) (map[EventKind]int, time.Time)
```
Counters returns a copy of the named pool's counts by event kind and the
time of the last event counted for it; a pool that has reported nothing has
neither. A pool reports asynchronously, so an event of an operation that has
just returned may not have been counted yet.


```go
func (t *StatusTracker) Pools() []string
```
Pools returns the names of the pools the tracker has counted an event for,
sorted. A pool that has reported nothing yet is not among them.


```go
func (t *StatusTracker) Subscribe(ctx context.Context) (<-chan struct{}, func())
```
Subscribe returns a channel signalled whenever any tracked pool's counters
change, and a function that must be called to release the subscription;
cancelling ctx releases it too, and releasing it twice is harmless. Signals
carry nothing and are coalesced, so a reader takes a fresh set of Counters.


```go
func (t *StatusTracker) Track(ctx context.Context) Option
```
Track returns the option that connects a pool to the tracker, to be passed
to New alongside the pool's other options:

    pool := vmspool.New(provider, "builders", tracker.Track(ctx), vmspool.WithSize(4))

Each call creates a channel of its own, counted from here rather than from
when the pool is created, so nothing a pool reports as it fills is missed;
Close releases a channel whose option never reached a pool.

It registers that channel with WithStatus, which holds one, so a pool given
both reports to whichever was applied last. A caller that wants both keeps
its own channel and feeds the tracker with Watch.


```go
func (t *StatusTracker) Watch(ctx context.Context, ch <-chan Event)
```
Watch counts the events arriving on ch until Close ends the tracker,
for a caller that has a status channel already; Track is the usual way to
connect a pool. Closing ch ends the forwarding of that channel alone and
takes nothing out of the counters.




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
func (v *VM) CompletionQueue() *CompletionQueue
```
CompletionQueue returns the completion queue configured for the pool the
VM came from, or nil if that pool has none, see WithCompletionQueue.
It is how work that has finished with a VM reaches the queue to park it in,
given nothing but the VM itself.


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
For the same reason only the first call to Delete replenishes the pool:
any further call finds the VM already deleted and does nothing.


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
func (v *VM) PoolName() string
```
PoolName returns the name of the pool to which the VM belongs.


```go
func (v *VM) Provider() Provider
```
Provider returns the provider used to create and manage the VM.


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
	VMResources
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

### Functions

```go
func ListVMs(ctx context.Context, providerNew ...ProviderNew) ([]VMInfo, error)
```
ListVMs lists all VMs from the given providers, sorted by VM name. It
returns a slice of VMInfo and any error encountered. Errors are accumulated
from all providers. Note, that this function is intentionally independent of
any pool and operates directly on the providers.




### Type VMResources
```go
type VMResources struct {
	DiskGiB  int `yaml:"disk_gib" doc:"size of the VM's disk in GiB; 0 leaves the image's own size"`
	NumCores int `yaml:"num_cores" doc:"number of CPU cores allocated to the VM; 0 leaves the image's own number"`
	MemGiB   int `yaml:"mem_gib" doc:"amount of RAM allocated to the VM in GiB; 0 leaves the image's own amount"`
}
```
VMResources describes the resources allocated to a VM. It carries
serialisation tags so that an application can embed it directly in the
configuration of a provider, rather than restating the same three figures;
a zero figure means unspecified, which a provider reports as unknown and
leaves to the image it creates VMs from.





