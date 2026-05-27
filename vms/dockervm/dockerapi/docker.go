// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package dockerapi implements cloudeng.io/vms.Instance using the Docker Go API.
package dockerapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	sdkclient "github.com/docker/go-sdk/client"
	mobycontainer "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/pkg/stdcopy"
	mobyclient "github.com/moby/moby/client"

	"cloudeng.io/os/executil"
	"cloudeng.io/vms"
)

// Instance implements vms.Instance backed by the Docker Go API.
// image is the Docker image to create containers from; name is the Docker container name.
type Instance struct {
	image string
	name  string
	opts  options
	logger *slog.Logger
	cli   sdkclient.SDKClient

	stateMu     sync.Mutex
	state       vms.State // GUARDED by stateMu
	ip          string    // GUARDED by stateMu
	containerID string    // GUARDED by stateMu

	// opMutex serialises Clone, Start, Stop, and Delete operations.
	opMutex sync.Mutex
}

// Option represents an option to New.
type Option func(o *options)

type options struct {
	pollingInterval time.Duration
	stopTimeout     time.Duration
	createArgs      []string // extra args interpreted as env vars (KEY=VAL) or labels
	containerCmd    []string
	logger          *slog.Logger
}

const (
	DefaultPollingInterval = 200 * time.Millisecond
	// DefaultStopTimeout is the graceful shutdown timeout for docker stop.
	DefaultStopTimeout = 10 * time.Second
)

// WithPollingInterval sets the interval used when polling for state transitions.
func WithPollingInterval(interval time.Duration) Option {
	return func(o *options) {
		o.pollingInterval = interval
	}
}

// WithStopTimeout sets the graceful shutdown timeout.
func WithStopTimeout(d time.Duration) Option {
	return func(o *options) {
		o.stopTimeout = d
	}
}

// WithContainerCmd overrides the default container command.
func WithContainerCmd(cmd ...string) Option {
	return func(o *options) {
		o.containerCmd = cmd
	}
}

// WithLogger sets the structured logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// DefaultContainerCmd returns the default command used to keep a container alive.
func DefaultContainerCmd() []string {
	return []string{"tail", "-f", "/dev/null"}
}

// New returns an Instance in StateInitial.
func New(ctx context.Context, image, name string, opts ...Option) (*Instance, error) {
	o := options{
		pollingInterval: DefaultPollingInterval,
		stopTimeout:     DefaultStopTimeout,
		containerCmd:    DefaultContainerCmd(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	cli, err := sdkclient.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	logger := o.logger
	if logger == nil {
		logger = slog.Default().With("module", "dockerapi", "image", image, "name", name)
	}
	return &Instance{
		image:  image,
		name:   name,
		state:  vms.StateInitial,
		opts:   o,
		logger: logger,
		cli:    cli,
	}, nil
}

// ID returns the container name.
func (inst *Instance) ID() string { return inst.name }

// Suspendable returns false; Docker containers do not support suspend.
func (inst *Instance) Suspendable() bool { return false }

func (inst *Instance) setState(state vms.State) vms.State {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	prev := inst.state
	inst.state = state
	return prev
}

func (inst *Instance) isActionAllowed(action vms.Action) (vms.State, bool) {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	return inst.state, inst.state.Allowed(action)
}

func (inst *Instance) getIP() string {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	return inst.ip
}

func (inst *Instance) clearIP() {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	inst.ip = ""
}

func (inst *Instance) getContainerID() string {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	return inst.containerID
}

// State returns the current state of the container.
func (inst *Instance) State(_ context.Context) vms.State {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	return inst.state
}

// Clone creates a Docker container from the image without starting it,
// transitioning from Initial/Deleted to Stopped.
func (inst *Instance) Clone(ctx context.Context) error {
	inst.opMutex.Lock()
	defer inst.opMutex.Unlock()

	if s, allowed := inst.isActionAllowed(vms.ActionClone); !allowed {
		return fmt.Errorf("action %s not allowed in state %s", vms.ActionClone, s)
	}
	prev := inst.setState(vms.StateCloning)

	trueVal := true
	createOpts := mobyclient.ContainerCreateOptions{
		Name:  inst.name,
		Image: inst.image,
		Config: &mobycontainer.Config{
			Cmd: inst.opts.containerCmd,
		},
		HostConfig: &mobycontainer.HostConfig{
			Init: &trueVal,
		},
	}

	inst.logger.Info("docker API: creating container", "name", inst.name, "image", inst.image)
	result, err := inst.cli.ContainerCreate(ctx, createOpts)
	if err != nil {
		inst.logger.Info("docker API: container create failed", "name", inst.name, "error", err)
		inst.setState(prev)
		return convertAPIError(inst.name, "create", err)
	}

	inst.stateMu.Lock()
	inst.containerID = result.ID
	inst.state = vms.StateStopped
	inst.stateMu.Unlock()
	inst.logger.Info("docker API: container created", "name", inst.name, "id", result.ID[:12])
	return nil
}

// Start starts the container and blocks until it is running.
func (inst *Instance) Start(ctx context.Context, stdout, stderr io.Writer) error {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	inst.opMutex.Lock()
	defer inst.opMutex.Unlock()

	if s, allowed := inst.isActionAllowed(vms.ActionStart); !allowed {
		return fmt.Errorf("action %s not allowed in state %s", vms.ActionStart, s)
	}
	prev := inst.setState(vms.StateStarting)
	cid := inst.getContainerID()

	inst.logger.Info("docker API: starting container", "name", inst.name)
	_, err := inst.cli.ContainerStart(ctx, cid, mobyclient.ContainerStartOptions{})
	if err != nil {
		inst.logger.Info("docker API: container start failed", "name", inst.name, "error", err)
		inst.setState(prev)
		return convertAPIError(inst.name, "start", err)
	}

	if err := inst.waitForDockerStatus(ctx, "running"); err != nil {
		_, _ = inst.cli.ContainerStop(context.Background(), cid, mobyclient.ContainerStopOptions{})
		inst.setState(prev)
		return fmt.Errorf("docker start %s: waiting for running state: %w", inst.name, err)
	}

	ip, err := inst.fetchContainerIP(ctx, cid)
	if err != nil {
		_, _ = inst.cli.ContainerStop(context.Background(), cid, mobyclient.ContainerStopOptions{})
		inst.setState(prev)
		return fmt.Errorf("docker start %s: fetching IP: %w", inst.name, err)
	}

	inst.stateMu.Lock()
	inst.ip = ip
	inst.state = vms.StateRunning
	inst.stateMu.Unlock()
	inst.logger.Info("docker API: container started", "name", inst.name, "ip", ip)
	return nil
}

// Stop stops the container.
func (inst *Instance) Stop(ctx context.Context, timeout time.Duration) (runErr, stopErr error) {
	inst.opMutex.Lock()
	defer inst.opMutex.Unlock()

	if s, allowed := inst.isActionAllowed(vms.ActionStop); !allowed {
		return nil, fmt.Errorf("action %s not allowed in state %s", vms.ActionStop, s)
	}
	prev := inst.setState(vms.StateStopping)
	if prev == vms.StateStopped {
		inst.setState(vms.StateStopped)
		return nil, nil
	}

	stopTimeout := inst.opts.stopTimeout
	if stopTimeout <= 0 {
		stopTimeout = DefaultStopTimeout
	}
	_ = timeout // context carries the caller's overall deadline
	timeoutSecs := int(stopTimeout.Seconds())
	cid := inst.getContainerID()

	inst.logger.Info("docker API: stopping container", "name", inst.name, "timeout", stopTimeout)
	_, err := inst.cli.ContainerStop(ctx, cid, mobyclient.ContainerStopOptions{
		Timeout: &timeoutSecs,
	})
	if err != nil {
		if isAlreadyStoppedAPIError(err) {
			inst.clearIP()
			inst.setState(vms.StateStopped)
			return nil, nil
		}
		inst.logger.Info("docker API: container stop failed", "name", inst.name, "error", err)
		inst.setState(vms.StateErrorUnknown)
		return nil, convertAPIError(inst.name, "stop", err)
	}
	inst.logger.Info("docker API: container stopped", "name", inst.name)
	inst.clearIP()
	inst.setState(vms.StateStopped)
	return nil, nil
}

// Suspend is not supported for Docker containers.
func (inst *Instance) Suspend(_ context.Context) error {
	return fmt.Errorf("docker containers do not support suspend")
}

// Delete removes the container.
func (inst *Instance) Delete(ctx context.Context) error {
	inst.opMutex.Lock()
	defer inst.opMutex.Unlock()

	if s, allowed := inst.isActionAllowed(vms.ActionDelete); !allowed {
		return fmt.Errorf("action %s not allowed in state %s", vms.ActionDelete, s)
	}
	prev := inst.setState(vms.StateDeleting)
	cid := inst.getContainerID()

	inst.logger.Info("docker API: removing container", "name", inst.name)
	_, err := inst.cli.ContainerRemove(ctx, cid, mobyclient.ContainerRemoveOptions{Force: true})
	if err != nil {
		if isNotFoundAPIError(err) {
			inst.setState(vms.StateDeleted)
			return nil
		}
		inst.logger.Info("docker API: container remove failed", "name", inst.name, "error", err)
		inst.setState(prev)
		return convertAPIError(inst.name, "remove", err)
	}
	inst.logger.Info("docker API: container removed", "name", inst.name)
	inst.setState(vms.StateDeleted)
	return nil
}

// Exec runs a command inside the running container with output written to stdout and stderr.
func (inst *Instance) Exec(ctx context.Context, stdout, stderr io.Writer, cmdStr string, args ...string) error {
	if state := inst.State(ctx); state != vms.StateRunning {
		return fmt.Errorf("exec only available for running containers, current state: %s", state)
	}
	cmd := append([]string{cmdStr}, args...)
	cid := inst.getContainerID()

	execID, err := inst.cli.ExecCreate(ctx, cid, mobyclient.ExecCreateOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		return fmt.Errorf("docker exec create %s: %w", inst.name, err)
	}

	hijack, err := inst.cli.ExecAttach(ctx, execID.ID, mobyclient.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("docker exec attach %s: %w", inst.name, err)
	}
	defer hijack.Close()

	if _, err := stdcopy.StdCopy(stdout, stderr, hijack.Reader); err != nil {
		return fmt.Errorf("docker exec read %s: %w", inst.name, err)
	}

	inspect, err := inst.cli.ExecInspect(ctx, execID.ID, mobyclient.ExecInspectOptions{})
	if err != nil {
		return fmt.Errorf("docker exec inspect %s: %w", inst.name, err)
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("docker exec %s: exit code %d", inst.name, inspect.ExitCode)
	}
	return nil
}

// Properties returns the container's IP address and clone metadata.
func (inst *Instance) Properties(_ context.Context) (vms.Properties, error) {
	ip := inst.getIP()
	return vms.Properties{
		IP:        ip,
		CloneInfo: CloneInfo{Image: inst.image, Name: inst.name},
	}, nil
}

func (inst *Instance) waitForDockerStatus(ctx context.Context, status string) error {
	cid := inst.getContainerID()
	found := func(ctx context.Context) (bool, error) {
		result, err := inst.cli.ContainerInspect(ctx, cid, mobyclient.ContainerInspectOptions{})
		if err != nil {
			if isNotFoundAPIError(err) {
				return true, fmt.Errorf("container %q not found", inst.name)
			}
			return true, err
		}
		if result.Container.State == nil {
			return false, nil
		}
		return string(result.Container.State.Status) == status, nil
	}
	return executil.WaitFor(ctx, inst.opts.pollingInterval, found)
}

func (inst *Instance) fetchContainerIP(ctx context.Context, cid string) (string, error) {
	result, err := inst.cli.ContainerInspect(ctx, cid, mobyclient.ContainerInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspect %s: %w", inst.name, err)
	}
	if result.Container.NetworkSettings == nil {
		return "", nil
	}
	for _, nw := range result.Container.NetworkSettings.Networks {
		if nw != nil && nw.IPAddress.IsValid() {
			return nw.IPAddress.String(), nil
		}
	}
	return "", nil
}

func isNotFoundAPIError(err error) bool {
	return errdefs.IsNotFound(err)
}

func isAlreadyStoppedAPIError(err error) bool {
	return errdefs.IsNotModified(err)
}



func convertAPIError(name, op string, err error) error {
	if isNotFoundAPIError(err) {
		return fmt.Errorf("docker %s %s: container does not exist: %w", op, name, vms.ErrVMNotFound)
	}
	if isAlreadyStoppedAPIError(err) {
		return fmt.Errorf("docker %s %s: container is not running: %w", op, name, vms.ErrVMNotRunning)
	}
	return fmt.Errorf("docker %s %s: %w", op, name, err)
}

// CloneInfo holds the image and container name used when creating the instance.
type CloneInfo struct {
	Image string
	Name  string
}

func (c CloneInfo) String() string {
	return fmt.Sprintf("image=%s name=%s", c.Image, c.Name)
}
