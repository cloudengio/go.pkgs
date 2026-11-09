// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"context"
	"errors"
	"time"

	"cloudeng.io/vms"
)

// ProviderDetails defines an interface for provider-specific details about the
// infrastructure implementing the VM provider.
type ProviderDetails interface {
	// Image returns the image the provider's VMs are created from, or an empty
	// string if the provider has no single image or does not know it.
	Image() string
}

// VMInfo is a backend-neutral summary of a VM managed by a Provider. It holds
// only the lightweight fields that are cheap to obtain in bulk via
// Provider.List.
type VMInfo struct {
	Name     string
	Pool     string
	State    string // backend-specific state string, e.g. "running", "stopped"
	Running  bool
	Accessed time.Time // best-effort last activity time; may be creation time if last access is unavailable
}

// VMDetail extends VMInfo with the fuller, potentially more expensive per-VM
// details returned by Provider.Get, such as the resources allocated to the VM.
type VMDetail struct {
	VMInfo
	DiskGiB  int // size of the VM's disk in GiB
	NumCores int // number of CPU cores allocated to the VM
	MemGiB   int // amount of RAM allocated to the VM in GiB
}

// Provider creates and manages the VMs for a Pool. In addition to constructing
// new instances it can enumerate, inspect and delete the VMs it has created,
// which pools and cleanup tooling use for status reporting and reclaiming
// orphaned VMs.
type Provider interface {
	// Name returns the name of the provider, typically the infrastructure
	// used to implement the pool such as 'tart' or 'docker'.
	Name() string

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

// ErrNoProvider should be returned by any operation that requires a VM provider
// when no appropriate provider has been configured.
var ErrNoProvider = errors.New("no VM provider configured")
