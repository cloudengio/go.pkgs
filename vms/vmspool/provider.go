// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"context"
	"errors"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/ociimage"
)

// ProviderDetails defines an interface for provider-specific details about the
// infrastructure implementing the VM provider.
type ProviderDetails interface {
	// Image returns the reference the provider's VMs are created from, or the
	// zero Reference if the provider has no single image or does not know it,
	// see ociimage.Reference.IsZero. It is parsed rather than a string so that
	// a caller naming something after the image, or deciding whether it can be
	// pulled, does not have to take the reference apart again.
	Image() ociimage.Reference
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

// VMResources describes the resources allocated to a VM. It carries
// serialisation tags so that an application can embed it directly in the
// configuration of a provider, rather than restating the same three figures;
// a zero figure means unspecified, which a provider reports as unknown and
// leaves to the image it creates VMs from.
type VMResources struct {
	DiskGiB  int `yaml:"disk_gib" doc:"size of the VM's disk in GiB; 0 leaves the image's own size"`
	NumCores int `yaml:"num_cores" doc:"number of CPU cores allocated to the VM; 0 leaves the image's own number"`
	MemGiB   int `yaml:"mem_gib" doc:"amount of RAM allocated to the VM in GiB; 0 leaves the image's own amount"`
}

// VMDetail extends VMInfo with the fuller, potentially more expensive per-VM
// details returned by Provider.Get, such as the resources allocated to the VM.
type VMDetail struct {
	VMInfo
	VMResources
}

// Provider creates and manages the VMs for a Pool. In addition to constructing
// new instances it can enumerate, inspect and delete the VMs it has created,
// which pools and cleanup tooling use for status reporting and reclaiming
// orphaned VMs.
type Provider interface {
	// Name returns the name of the provider, typically the infrastructure
	// used to implement the pool such as 'tart' or 'docker'.
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

// ErrNoProvider should be returned by any operation that requires a VM provider
// when no appropriate provider has been configured.
var ErrNoProvider = errors.New("no VM provider configured")
