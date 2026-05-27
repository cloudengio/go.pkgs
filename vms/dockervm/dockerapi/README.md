# Package [cloudeng.io/vms/dockervm/dockerapi](https://pkg.go.dev/cloudeng.io/vms/dockervm/dockerapi?tab=doc)

```go
import cloudeng.io/vms/dockervm/dockerapi
```

Package dockerapi implements cloudeng.io/vms.Instance using the Docker Go
API.

## Constants
### DefaultPollingInterval, DefaultForceStopTimeout
```go
DefaultPollingInterval = 200 * time.Millisecond
// DefaultForceStopTimeout is the graceful shutdown timeout for docker stop.
DefaultForceStopTimeout = 10 * time.Second

```



## Functions
### Func DefaultContainerCmd
```go
func DefaultContainerCmd() []string
```
DefaultContainerCmd returns the default command used to keep a container
alive.



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




### Type ContainerInfo
```go
type ContainerInfo struct {
	Name            string
	State           ContainerStateInfo
	NetworkSettings ContainerNetworkInfo
}
```
ContainerInfo holds the parsed result of a Docker container inspection.

### Functions

```go
func InspectContainer(ctx context.Context, name string) (ContainerInfo, bool, error)
```
InspectContainer queries the Docker daemon for the named container using the
Docker API. Returns (zero, false, nil) if the container does not exist.


```go
func InspectContainerClient(ctx context.Context, client sdkclient.SDKClient, name string) (ContainerInfo, bool, error)
```
InspectContainerClient queries the Docker daemon using the supplied client
for the named container using the Docker API. Returns (zero, false, nil) if
the container does not exist.



### Methods

```go
func (c ContainerInfo) VMSState() vms.State
```
VMSState maps the Docker container status to a vms.State.




### Type ContainerNetworkInfo
```go
type ContainerNetworkInfo struct {
	IPAddress string
}
```
ContainerNetworkInfo represents the primary network information for a
container.


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
ContainerStateInfo represents the state portion of a Docker container
inspection.


### Type Instance
```go
type Instance struct {
	// contains filtered or unexported fields
}
```
Instance implements vms.Instance backed by the Docker Go API. image is the
Docker image to create containers from; name is the Docker container name.

### Functions

```go
func New(ctx context.Context, image, name string, opts ...Option) (*Instance, error)
```
New returns an Instance in StateInitial.



### Methods

```go
func (inst *Instance) Clone(ctx context.Context) error
```
Clone creates a Docker container from the image without starting it,
transitioning from Initial/Deleted to Stopped.


```go
func (inst *Instance) Delete(ctx context.Context) error
```
Delete removes the container.


```go
func (inst *Instance) Exec(ctx context.Context, stdout, stderr io.Writer, cmdStr string, args ...string) error
```
Exec runs a command inside the running container with output written to
stdout and stderr.


```go
func (inst *Instance) ID() string
```
ID returns the container name.


```go
func (inst *Instance) Properties(_ context.Context) (vms.Properties, error)
```
Properties returns the container's IP address and clone metadata.


```go
func (inst *Instance) Start(ctx context.Context, _, _ io.Writer) error
```
Start starts the container and blocks until it is running.


```go
func (inst *Instance) State(_ context.Context) vms.State
```
State returns the current state of the container.


```go
func (inst *Instance) Stop(ctx context.Context, timeout time.Duration) (runErr, stopErr error)
```
Stop stops the container.


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
func WithCreateEnv(env []string) Option
```
WithCreateEnv sets extra args interpreted as env vars (KEY=VAL) or labels.


```go
func WithForceStopTimeout(d time.Duration) Option
```
WithForceStopTimeout sets the graceful shutdown timeout.


```go
func WithLogger(logger *slog.Logger) Option
```
WithLogger sets the structured logger.


```go
func WithPollingInterval(interval time.Duration) Option
```
WithPollingInterval sets the interval used when polling for state
transitions.







