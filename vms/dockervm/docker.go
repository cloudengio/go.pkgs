// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package dockervm implements cloudeng.io/vms.Instance using the Docker CLI.
package dockervm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloudeng.io/os/executil"
	"cloudeng.io/vms"
)

// Instance implements vms.Instance backed by the Docker CLI.
// image is the Docker image to create containers from;
// name is the Docker container name.
type Instance struct {
	image string
	name  string
	opts  options
	logger *slog.Logger

	stateMu sync.Mutex
	state   vms.State // GUARDED by stateMu
	ip      string    // GUARDED by stateMu

	// opMutex serialises Clone, Start, Stop, and Delete operations.
	opMutex sync.Mutex
}

// Option represents an option to New.
type Option func(o *options)

type options struct {
	pollingInterval  time.Duration
	stopTimeout      time.Duration // graceful shutdown timeout passed to docker stop --timeout
	createArgs       []string      // extra args to "docker create"
	containerCmd     []string      // the command run inside the container
	logger           *slog.Logger
}

const (
	DefaultPollingInterval = 200 * time.Millisecond
	// DefaultStopTimeout is the graceful shutdown timeout for docker stop.
	// After this period Docker sends SIGKILL. Kept short because containers
	// should not need long graceful shutdown windows.
	DefaultStopTimeout = 10 * time.Second
)

// WithPollingInterval sets the interval used when polling for state transitions.
func WithPollingInterval(interval time.Duration) Option {
	return func(o *options) {
		o.pollingInterval = interval
	}
}

// WithCreateArgs appends extra arguments to the "docker create" command.
// Useful for setting environment variables, volume mounts, network settings, etc.
func WithCreateArgs(args ...string) Option {
	return func(o *options) {
		o.createArgs = append(o.createArgs, args...)
	}
}

// WithStopTimeout sets the graceful shutdown timeout passed to "docker stop --timeout".
// After this period Docker sends SIGKILL. Defaults to DefaultStopTimeout.
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

// WithLogger sets the structured logger used for command tracing.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// DefaultContainerCmd returns the default command used to keep a container alive.
// tail -f /dev/null is used because it handles SIGTERM properly (unlike sleep infinity
// on some BusyBox implementations).
func DefaultContainerCmd() []string {
	return []string{"tail", "-f", "/dev/null"}
}

// New returns an Instance in StateInitial.
// image is the Docker image to use; name is the container name.
func New(_ context.Context, image, name string, opts ...Option) *Instance {
	o := options{
		pollingInterval: DefaultPollingInterval,
		stopTimeout:     DefaultStopTimeout,
		containerCmd:    DefaultContainerCmd(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	logger := o.logger
	if logger == nil {
		logger = slog.Default().With("module", "docker", "image", image, "name", name)
	}
	return &Instance{
		image:  image,
		name:   name,
		state:  vms.StateInitial,
		opts:   o,
		logger: logger,
	}
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

// State returns the current state of the container.
func (inst *Instance) State(_ context.Context) vms.State {
	inst.stateMu.Lock()
	defer inst.stateMu.Unlock()
	return inst.state
}

// Clone runs "docker create --name <name> [createArgs] <image> [containerCmd]"
// and transitions the instance from Initial/Deleted to Stopped.
func (inst *Instance) Clone(ctx context.Context) error {
	inst.opMutex.Lock()
	defer inst.opMutex.Unlock()

	if s, allowed := inst.isActionAllowed(vms.ActionClone); !allowed {
		return fmt.Errorf("action %s not allowed in state %s", vms.ActionClone, s)
	}
	prev := inst.setState(vms.StateCloning)

	// --init adds tini as PID 1 to ensure proper signal forwarding to the container process.
	args := []string{"create", "--init", "--name", inst.name}
	args = append(args, inst.opts.createArgs...)
	args = append(args, inst.image)
	args = append(args, inst.opts.containerCmd...)

	inst.logger.Info("docker command issued", "args", args)
	stderrBuf := executil.NewTailWriter(1024)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := string(stderrBuf.Bytes())
		inst.logger.Info("docker command failed", "args", args, "stderr", stderr, "error", err)
		inst.setState(prev)
		return convertError(args, stderr, err)
	}
	inst.logger.Info("docker command completed", "args", args)
	inst.setState(vms.StateStopped)
	return nil
}

// Start runs "docker start <name>" and blocks until the container is running.
// The stdout and stderr writers receive the docker start command output;
// the container's own stdout/stderr are managed by the Docker daemon.
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

	args := []string{"start", inst.name}
	inst.logger.Info("docker command issued", "args", args)

	stderrBuf := executil.NewTailWriter(1024)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = stdout
	cmd.Stderr = io.MultiWriter(stderr, stderrBuf)
	if err := cmd.Run(); err != nil {
		stderrStr := string(stderrBuf.Bytes())
		inst.logger.Info("docker command failed", "args", args, "stderr", stderrStr, "error", err)
		inst.setState(prev)
		return convertError(args, stderrStr, err)
	}

	if err := inst.waitForDockerStatus(ctx, "running"); err != nil {
		_ = exec.CommandContext(context.Background(), "docker", "stop", inst.name).Run()
		inst.setState(prev)
		return fmt.Errorf("docker start %s: waiting for running state: %w", inst.name, err)
	}

	ip, err := inst.fetchContainerIP(ctx)
	if err != nil {
		_ = exec.CommandContext(context.Background(), "docker", "stop", inst.name).Run()
		inst.setState(prev)
		return fmt.Errorf("docker start %s: fetching IP: %w", inst.name, err)
	}

	inst.stateMu.Lock()
	inst.ip = ip
	inst.state = vms.StateRunning
	inst.stateMu.Unlock()
	inst.logger.Info("docker command completed", "args", args, "ip", ip)
	return nil
}

// Stop runs "docker stop --timeout <seconds> <name>" and transitions to Stopped.
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

	// Use the instance's configured stop timeout (not the caller's timeout, which
	// is an overall operation deadline). Docker containers should stop quickly;
	// if needed callers can extend via WithStopTimeout.
	stopTimeout := inst.opts.stopTimeout
	if stopTimeout <= 0 {
		stopTimeout = DefaultStopTimeout
	}
	_ = timeout // the ctx carries the caller's overall deadline
	args := []string{"stop", "--timeout", strconv.Itoa(int(stopTimeout.Seconds())), inst.name}

	inst.logger.Info("docker command issued", "args", args)
	stderrBuf := executil.NewTailWriter(1024)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := string(stderrBuf.Bytes())
		inst.logger.Info("docker command failed", "args", args, "stderr", stderr, "error", err)
		if isAlreadyStoppedError(stderr) {
			inst.clearIP()
			inst.setState(vms.StateStopped)
			return nil, nil
		}
		inst.setState(vms.StateErrorUnknown)
		return nil, convertError(args, stderr, err)
	}
	inst.logger.Info("docker command completed", "args", args)
	inst.clearIP()
	inst.setState(vms.StateStopped)
	return nil, nil
}

// Suspend is not supported for Docker containers.
func (inst *Instance) Suspend(_ context.Context) error {
	return fmt.Errorf("docker containers do not support suspend")
}

// Delete runs "docker rm --force <name>" and transitions to Deleted.
func (inst *Instance) Delete(ctx context.Context) error {
	inst.opMutex.Lock()
	defer inst.opMutex.Unlock()

	if s, allowed := inst.isActionAllowed(vms.ActionDelete); !allowed {
		return fmt.Errorf("action %s not allowed in state %s", vms.ActionDelete, s)
	}
	prev := inst.setState(vms.StateDeleting)

	args := []string{"rm", "--force", inst.name}
	inst.logger.Info("docker command issued", "args", args)
	stderrBuf := executil.NewTailWriter(1024)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := string(stderrBuf.Bytes())
		inst.logger.Info("docker command failed", "args", args, "stderr", stderr, "error", err)
		if isContainerNotFound(stderr) {
			inst.setState(vms.StateDeleted)
			return nil
		}
		inst.setState(prev)
		return convertError(args, stderr, err)
	}
	inst.logger.Info("docker command completed", "args", args)
	inst.setState(vms.StateDeleted)
	return nil
}

// Exec runs "docker exec <name> <cmd> <args...>" with output connected to
// the provided writers.
func (inst *Instance) Exec(ctx context.Context, stdout, stderr io.Writer, cmdStr string, args ...string) error {
	if state := inst.State(ctx); state != vms.StateRunning {
		return fmt.Errorf("exec only available for running containers, current state: %s", state)
	}
	allArgs := append([]string{"exec", inst.name, cmdStr}, args...)
	c := exec.CommandContext(ctx, "docker", allArgs...)
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", strings.Join(allArgs, " "), err)
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
	found := func(ctx context.Context) (bool, error) {
		info, exists, err := InspectContainer(ctx, inst.name)
		if err != nil {
			return true, err
		}
		if !exists {
			return true, fmt.Errorf("container %q not found", inst.name)
		}
		return info.State.Status == status, nil
	}
	return executil.WaitFor(ctx, inst.opts.pollingInterval, found)
}

func (inst *Instance) fetchContainerIP(ctx context.Context) (string, error) {
	info, found, err := InspectContainer(ctx, inst.name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("container %q not found", inst.name)
	}
	return info.NetworkSettings.IPAddress, nil
}

var (
	reContainerNotFound = regexp.MustCompile(`(?i)no such (container|object)`)
	reContainerStopped  = regexp.MustCompile(`(?i)is not running`)
)

func isContainerNotFound(stderr string) bool {
	return reContainerNotFound.MatchString(stderr)
}

func isAlreadyStoppedError(stderr string) bool {
	return reContainerStopped.MatchString(stderr)
}

func convertError(args []string, stderr string, err error) error {
	cl := strings.Join(args, " ")
	if isContainerNotFound(stderr) {
		return fmt.Errorf("%s: container does not exist: %s: %v: %w", cl, stderr, err, vms.ErrVMNotFound)
	}
	if isAlreadyStoppedError(stderr) {
		return fmt.Errorf("%s: container is not running: %s: %v: %w", cl, stderr, err, vms.ErrVMNotRunning)
	}
	return fmt.Errorf("%s: %s: %w", cl, stderr, err)
}

// CloneInfo holds the image and container name used when creating the instance.
type CloneInfo struct {
	Image string
	Name  string
}

func (c CloneInfo) String() string {
	return fmt.Sprintf("image=%s name=%s", c.Image, c.Name)
}
