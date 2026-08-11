# Package [cloudeng.io/vms/dockervm](https://pkg.go.dev/cloudeng.io/vms/dockervm?tab=doc)

```go
import cloudeng.io/vms/dockervm
```

Package dockervm implements cloudeng.io/vms.Instance using the Docker CLI.

## Constants
### DefaultPollingInterval, DefaultForceStopTimeout
```go
DefaultPollingInterval = 200 * time.Millisecond
DefaultForceStopTimeout = 10 * time.Second

```



## Functions
### Func DefaultContainerCmd
```go
func DefaultContainerCmd() []string
```
DefaultContainerCmd returns the default command used to keep a container
alive. tail -f /dev/null is used because it handles SIGTERM properly (unlike
sleep infinity on some BusyBox implementations).



## Types
### Type CloneInfo
```go
type CloneInfo struct {
	Image string
	Name  string
}
```
CloneInfo holds the image and container name used when creating the
instance.

### Methods

```go
func (c CloneInfo) String() string
```




### Type Constructor
```go
type Constructor = func(ctx context.Context) vms.Instance
```
Constructor creates a new, uninitialized Docker VM instance. Each call must
return a distinct vms.Instance (typically via New with a unique name).
ctx governs any work done to construct the instance.


### Type ContainerInfo
```go
type ContainerInfo struct {
	Name            string
	State           ContainerStateInfo
	NetworkSettings ContainerNetworkInfo
}
```
ContainerInfo holds the parsed output from "docker inspect <name>".

### Functions

```go
func InspectContainer(ctx context.Context, name string) (ContainerInfo, bool, error)
```
InspectContainer runs "docker inspect <name>" and returns the parsed result.
Returns (zero, false, nil) if the container does not exist.



### Methods

```go
func (c ContainerInfo) VMSState() vms.State
```
VMSState maps the docker container status to a vms.State.




### Type ContainerNetworkInfo
```go
type ContainerNetworkInfo struct {
	IPAddress string
}
```
ContainerNetworkInfo represents the network information for a container.


### Type ContainerStateInfo
```go
type ContainerStateInfo struct {
	Status     string // "created", "running", "paused", "restarting", "removing", "exited", "dead"
	Running    bool
	Paused     bool
	Restarting bool
	Dead       bool
	ExitCode   int
}
```
ContainerStateInfo represents the State portion of docker inspect output.


### Type Instance
```go
type Instance struct {
	// contains filtered or unexported fields
}
```
Instance implements vms.Instance backed by the Docker CLI. image is the
Docker image to create containers from; name is the Docker container name.

### Functions

```go
func New(_ context.Context, image, name string, opts ...Option) *Instance
```
New returns an Instance in StateInitial. image is the Docker image to use;
name is the container name.



### Methods

```go
func (inst *Instance) Clone(ctx context.Context) error
```
Clone runs "docker create --name <name> [createArgs] <image> [containerCmd]"
and transitions the instance from Initial/Deleted to Stopped.


```go
func (inst *Instance) Delete(ctx context.Context) error
```
Delete runs "docker rm --force <name>" and transitions to Deleted.


```go
func (inst *Instance) Exec(ctx context.Context, stdout, stderr io.Writer, cmdStr string, args ...string) error
```
Exec runs "docker exec <name> <cmd> <args...>" with output connected to the
provided writers.


```go
func (inst *Instance) ID() string
```
ID returns the container name.


```go
func (inst *Instance) Properties(_ context.Context) (vms.Properties, error)
```
Properties returns the container's IP address and clone metadata.


```go
func (inst *Instance) Start(ctx context.Context, stdout, stderr io.Writer) error
```
Start runs "docker start <name>" and blocks until the container is running.
The stdout and stderr writers receive the docker start command output;
the container's own stdout/stderr are managed by the Docker daemon.


```go
func (inst *Instance) State(_ context.Context) vms.State
```
State returns the current state of the container.


```go
func (inst *Instance) Stop(ctx context.Context, timeout time.Duration) (runErr, stopErr error)
```
Stop runs "docker stop --timeout <seconds> <name>" and transitions to
Stopped.


```go
func (inst *Instance) Suspend(_ context.Context) error
```
Suspend is not supported for Docker containers.


```go
func (inst *Instance) Suspendable() bool
```
Suspendable returns false; Docker containers do not support suspend.




### Type Option
```go
type Option func(o *options)
```
Option represents an option to New.

### Functions

```go
func WithContainerCmd(cmd ...string) Option
```
WithContainerCmd overrides the default container command.


```go
func WithCreateArgs(args ...string) Option
```
WithCreateArgs appends extra arguments to the "docker create" command.
Useful for setting environment variables, volume mounts, network settings,
etc.


```go
func WithForceStopTimeout(d time.Duration) Option
```
WithForceStopTimeout sets the graceful shutdown timeout passed to "docker
stop --timeout". After this period Docker sends SIGKILL. Defaults to
DefaultForceStopTimeout.


```go
func WithLogger(logger *slog.Logger) Option
```
WithLogger sets the structured logger used for command tracing.


```go
func WithPollingInterval(interval time.Duration) Option
```
WithPollingInterval sets the interval used when polling for state
transitions.




### Type Provider
```go
type Provider struct {
	// contains filtered or unexported fields
}
```
Provider is a vmspool.Provider backed by Docker containers. It delegates VM
construction to a caller-supplied Constructor and implements List, Get and
Delete directly via the docker CLI, so using Docker with a vmspool.Pool only
requires supplying the construction function.

### Functions

```go
func NewProvider(constructor Constructor, opts ...ProviderOption) *Provider
```
NewProvider returns a Provider that constructs containers with constructor
and implements the remaining vmspool.Provider methods via the docker CLI.



### Methods

```go
func (p *Provider) Delete(ctx context.Context, stopTimeout time.Duration) ([]string, error)
```
Delete implements vmspool.Provider, stopping (if running) and removing
every container belonging to this pool. Deletion continues past individual
failures.


```go
func (p *Provider) Get(ctx context.Context, vmName string) (vmspool.VMDetail, error)
```
Get implements vmspool.Provider, returning the resources allocated to a
single container via "docker inspect --size".


```go
func (p *Provider) List(ctx context.Context) ([]vmspool.VMInfo, error)
```
List implements vmspool.Provider.


```go
func (p *Provider) New(ctx context.Context) vms.Instance
```
New implements vmspool.Provider.




### Type ProviderOption
```go
type ProviderOption func(*Provider)
```
ProviderOption configures a Provider.

### Functions

```go
func WithNamePrefix(prefix string) ProviderOption
```
WithNamePrefix scopes List and Delete to containers whose names start
with prefix. Without it a Provider manages every container on the daemon,
which is rarely desirable when the host runs unrelated containers.


```go
func WithPoolName(name string) ProviderOption
```
WithPoolName sets the pool name reported in VMInfo.Pool for the Provider's
VMs.







