// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockervm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
)

// Constructor creates a new, uninitialized Docker VM instance. Each call must
// return a distinct vms.Instance (typically via New with a unique name). ctx
// governs any work done to construct the instance. It returns an error if the
// instance could not be created.
type Constructor = func(ctx context.Context) (vms.Instance, error)

// Provider is a vmspool.Provider backed by Docker containers. It delegates VM
// construction to a caller-supplied Constructor and implements List, Get and
// Delete directly via the docker CLI, so using Docker with a vmspool.Pool only
// requires supplying the construction function.
type Provider struct {
	constructor Constructor
	prefix      string // if set, only containers whose name has this prefix are managed
	pool        string // optional pool name used to tag VMInfo.Pool
	logger      *slog.Logger
}

var _ vmspool.Provider = (*Provider)(nil)

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithNamePrefix scopes List and Delete to containers whose names start with
// prefix. Without it a Provider manages every container on the daemon, which is
// rarely desirable when the host runs unrelated containers.
func WithNamePrefix(prefix string) ProviderOption {
	return func(p *Provider) { p.prefix = prefix }
}

// WithPoolName sets the pool name reported in VMInfo.Pool for the Provider's VMs.
func WithPoolName(name string) ProviderOption {
	return func(p *Provider) { p.pool = name }
}

// NewProvider returns a Provider that constructs containers with constructor and
// implements the remaining vmspool.Provider methods via the docker CLI.
func NewProvider(constructor Constructor, opts ...ProviderOption) *Provider {
	p := &Provider{
		constructor: constructor,
		logger:      slog.Default().With("module", "dockervm"),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// New implements vmspool.Provider.
func (p *Provider) New(ctx context.Context) (vms.Instance, error) { return p.constructor(ctx) }

// psEntry is one line of "docker ps --format {{json .}}".
type psEntry struct {
	Names     string
	State     string
	CreatedAt string
}

// list returns the containers the Provider manages, in any state.
func (p *Provider) list(ctx context.Context) ([]psEntry, error) {
	args := []string{"ps", "--all", "--no-trunc", "--format", "{{json .}}"}
	if p.prefix != "" {
		args = append(args, "--filter", "name="+p.prefix)
	}
	out, err := runDockerOut(ctx, args...)
	if err != nil {
		return nil, err
	}
	var entries []psEntry
	for line := range strings.Lines(strings.TrimSpace(string(out))) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e psEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("docker ps: parse %q: %w", line, err)
		}
		// The name filter matches substrings, so keep only containers that
		// actually use our prefix.
		e.Names = firstContainerName(e.Names)
		if p.prefix != "" && !strings.HasPrefix(e.Names, p.prefix) {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// firstContainerName returns the first, de-slashed name from a docker ps Names
// field (which may be comma-separated).
func firstContainerName(names string) string {
	name, _, _ := strings.Cut(names, ",")
	return strings.TrimPrefix(strings.TrimSpace(name), "/")
}

// List implements vmspool.Provider.
func (p *Provider) List(ctx context.Context) ([]vmspool.VMInfo, error) {
	entries, err := p.list(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]vmspool.VMInfo, 0, len(entries))
	for _, e := range entries {
		out = append(out, vmspool.VMInfo{
			Name:     e.Names,
			Pool:     p.pool,
			State:    e.State,
			Running:  e.State == "running",
			Accessed: parseDockerTime(e.CreatedAt),
		})
	}
	return out, nil
}

// Get implements vmspool.Provider, returning the resources allocated to a single
// container via "docker inspect --size".
func (p *Provider) Get(ctx context.Context, vmName string) (vmspool.VMDetail, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--size", "--format", "{{json .}}", vmName) //nolint:gosec // G204: vmName is an internal container name.
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if isContainerNotFound(stderr.String()) {
			return vmspool.VMDetail{}, fmt.Errorf("no such vm: %s", vmName)
		}
		return vmspool.VMDetail{}, fmt.Errorf("docker inspect %s: %s: %w", vmName, strings.TrimSpace(stderr.String()), err)
	}
	var d struct {
		Name       string
		State      struct{ Status string }
		HostConfig struct {
			NanoCpus int64
			Memory   int64
		}
		SizeRootFs int64
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &d); err != nil {
		return vmspool.VMDetail{}, fmt.Errorf("docker inspect %s: parse: %w", vmName, err)
	}
	return vmspool.VMDetail{
		VMInfo: vmspool.VMInfo{
			Name:    strings.TrimPrefix(d.Name, "/"),
			Pool:    p.pool,
			State:   d.State.Status,
			Running: d.State.Status == "running",
		},
		DiskGiB:  bytesToGiB(d.SizeRootFs),
		NumCores: int(d.HostConfig.NanoCpus / 1_000_000_000),
		MemGiB:   bytesToGiB(d.HostConfig.Memory),
	}, nil
}

// Delete implements vmspool.Provider, stopping (if running) and removing every
// container belonging to this pool. Deletion continues past individual failures.
func (p *Provider) Delete(ctx context.Context, stopTimeout time.Duration) ([]string, error) {
	entries, err := p.list(ctx)
	if err != nil {
		return nil, err
	}
	var errs errors.M
	deleted := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.State == "running" {
			args := []string{"stop"}
			if stopTimeout > 0 {
				args = append(args, "--timeout", strconv.Itoa(int(stopTimeout.Seconds())))
			}
			args = append(args, e.Names)
			if err := runDocker(ctx, args...); err != nil {
				p.logger.Error("failed to stop container", "container", e.Names, "err", err)
				errs.Append(err)
			}
		}
		if err := runDocker(ctx, "rm", "--force", e.Names); err != nil {
			p.logger.Error("failed to remove container", "container", e.Names, "err", err)
			errs.Append(err)
			continue
		}
		deleted = append(deleted, e.Names)
	}
	return deleted, errs.Err()
}

func bytesToGiB(b int64) int {
	const giB = 1 << 30
	if b <= 0 {
		return 0
	}
	return int((b + giB/2) / giB)
}

// parseDockerTime parses the "docker ps" CreatedAt field, returning the zero
// time if it cannot be parsed.
func parseDockerTime(s string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04:05 -0700 MST", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func runDocker(ctx context.Context, args ...string) error {
	_, err := runDockerOut(ctx, args...)
	return err
}

func runDockerOut(ctx context.Context, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", args...) //nolint:gosec // G204: args are internal, non-user values.
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr.String()), err)
	}
	return stdout.Bytes(), nil
}
