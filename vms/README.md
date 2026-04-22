# Package [cloudeng.io/vms](https://pkg.go.dev/cloudeng.io/vms?tab=doc)

```go
import cloudeng.io/vms
```

Package vms provides support for working with virtual machines. State
machine transitions for virtual machine instances. Each line lists a state
followed by the actions that can be applied and the resulting next state.
"(waiting)" denotes the ActionNone no-op used to poll until an in-progress
operation completes.

    Initial         Clone -> Cloning
    Starting        (waiting) -> Starting
    Running         Stop -> Stopping,  Suspend -> Suspending
    Stopping        (waiting) -> Stopping
    Stopped         Run -> Starting,  Stop -> Stopped,  Delete -> Deleting
    Suspending      (waiting) -> Suspending
    Suspended       Run -> Starting,  Suspend -> Suspended,  Delete -> Deleting
    Deleting        (waiting) -> Deleting
    Deleted         Clone -> Cloning
    State(10)       Delete -> Deleting

## Functions
### Func CleanupVM
```go
func CleanupVM(ctx context.Context, inst Instance) error
```
CleanupVM attempts to clean up the given instance by stopping and deleting
it if necessary. Suspended VMs are stopped before deletion. It returns an
error if any of the operations fail.

### Func PrintStates
```go
func PrintStates(out io.Writer)
```
PrintStates writes a human-readable description of every state and its valid
transitions to out.

### Func WaitForState
```go
func WaitForState(ctx context.Context, inst Instance, interval time.Duration, final State, intermediate ...State) error
```
WaitForStateermediate polls inst.State until it returns the requised
final state or the context is done. If intermediate states are provided,
it also checks that any intermediate states returned by inst.State are
in the set of allowed intermediate states on the way to the final state,
returning an error if an unexpected intermediate state is observed.



## Types
### Type Action
```go
type Action int
```
Action represents an operation that causes a state transition.

### Constants
### ActionNone, ActionClone, ActionStart, ActionStop, ActionSuspend, ActionDelete
```go
ActionNone Action = iota
ActionClone
ActionStart
ActionStop
ActionSuspend
ActionDelete

```



### Methods

```go
func (a Action) String() string
```




### Type Instance
```go
type Instance interface {

	// Clone prepares an instance for being stated. It should be
	// a synchronouse operation and hwne it returns the state should be Stopped.
	// States: success: [Initial] -> Cloning -> Stopped
	// States:   error: [Initial] -> Cloning -> Initial
	Clone(ctx context.Context) error

	// Start starts the instance. It returns once the instance is running.
	// States: success: [Stopped] -> Starting -> Running
	// States:   error: [Stopped] -> Starting -> StateErrorUnknown or Stopped
	Start(ctx context.Context, stdout, stderr io.Writer) error

	// Stop stops the instance. It returns once the instance is stopped.
	// The timeout parameter specifies how long to wait for a graceful shutdown
	// before forcefully shutting down the vm instance.
	// States: [Running, Stopped] -> Stopping -> Stopped
	// States: error: [Running] -> Stopping -> Stopped or StateErrorUnknown
	Stop(ctx context.Context, timeout time.Duration) (runErr, stopErr error)

	// Suspendable returns true if the instance supports being suspended.
	Suspendable() bool

	// Suspend suspends the instance. It returns once the instance is suspended.
	// States: [Running] -> Suspending -> Suspsended
	// States: error: [Running] -> Suspending -> Suspended or StateErrorUnknown
	Suspend(ctx context.Context) error

	// Delete deletes the instance.
	// States: [Stopped, Suspended, ErrorUnknown] -> Deleting -> Deleted
	Delete(ctx context.Context) error

	// State returns the current state of the instance, it may be
	// called at any time.
	State(ctx context.Context) State

	// Exec executes the given command in the instance and returns when the
	// command completes.
	// Exec does not alter the state of the instance.
	Exec(ctx context.Context, stdout, stderr io.Writer, cmd string, args ...string) error

	// Properties returns the properties of a running instance.
	Properties(ctx context.Context) (Properties, error)
}
```
Instance represents a virtual machine instance that can be managed through
a lifecycle of states. Operations change the state of the instance as
indicated below for successful operations. Error returning operations will
either leave the state unchange, or transition to StateErrorUnknown if the
state cannot be determined. Intgermediate states (eg. Stopping, Starting)
may be observed while the operation is in progress.


### Type Properties
```go
type Properties struct {
	IP string // The IP address of the instance, if available.

}
```
Properties represents the properties of a virtual machine instance.


### Type State
```go
type State int
```
State represents the state of a virtual machine instance.

### Constants
### StateInitial, StateCloning, StateStarting, StateRunning, StateStopping, StateStopped, StateSuspending, StateSuspended, StateDeleting, StateDeleted, StateErrorUnknown
```go
StateInitial State = iota
StateCloning
StateStarting
StateRunning
StateStopping
StateStopped
StateSuspending
StateSuspended
StateDeleting
StateDeleted
StateErrorUnknown

```



### Methods

```go
func (s State) Allowed(action Action) bool
```
Allowed returns true if the given action is valid from the current state.


```go
func (s State) String() string
```


```go
func (s State) Transition(action Action) (State, bool)
```
Transition returns the next State reached by applying action to from,
or an false if the transition is not valid.


```go
func (s State) ValidActions() []Action
```
ValidActions returns the set of actions that are valid from the given state.







