// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"cloudeng.io/errors"
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

// CachingProvider wraps a Provider and caches the per-VM details that are
// expensive to obtain, which for a provider backed by a command line tool
// means one invocation of it per VM. What it caches is the part of a VMDetail
// that does not change whilst a VM exists: the resources it was created with.
// The state of a VM does change, so GetAll never serves that from the cache,
// see the VMInfo rule there.
//
// A zero CachingProvider wrapping a Provider is ready to use, and it is safe
// for concurrent use. It is a Provider itself, so a pool can be created with
// one, and it caches nothing a pool itself asks for: a pool creates VMs and
// leaves reporting on them to its application.
type CachingProvider struct {
	Provider
	mu      sync.Mutex
	details map[string]VMDetail
}

var _ Provider = (*CachingProvider)(nil)

// GetAll returns the full details of every VM the wrapped Provider lists,
// reading the details of each VM it has not seen before and serving the rest
// from the cache. VMs the listing no longer reports are dropped from it, so it
// holds the VMs that exist rather than every VM the provider has ever had.
//
// The listing is fresher than anything cached, so each VMDetail carries the
// VMInfo the listing gave it: a cached VM reports the state it is in now, not
// the state it was in when its details were read.
//
// A VM whose details cannot be read is still returned, carrying what the
// listing said about it and the provider's default resources: a VM that
// exists and cannot be inspected is of more use reported than omitted, not
// least because the count of VMs would otherwise disagree with the pool's own.
// Its name is returned in failed, and nothing is cached for it, so the next
// call reads it again rather than leaving it reporting defaults for as long as
// it exists. The error accumulates those failures and is returned alongside
// results that are usable whether or not it is nil; only a listing that fails
// returns no results at all.
//
// The details are returned ordered by VM name, whatever order List reports.
func (c *CachingProvider) GetAll(ctx context.Context) ([]VMDetail, []string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// List is the wrapped provider's; only Get is cached.
	infos, err := c.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	cache := make(map[string]VMDetail, len(infos))
	out := make([]VMDetail, 0, len(infos))
	var failed []string
	var errs errors.M
	for _, info := range infos {
		detail, ok := c.details[info.Name]
		if !ok {
			read, err := c.getLocked(ctx, info.Name)
			if err != nil {
				failed = append(failed, info.Name)
				errs.Append(err)
				out = append(out, c.withDefaults(VMDetail{VMInfo: info}))
				continue
			}
			detail = read
		}
		// The listing is fresher than anything cached, so it wins.
		detail.VMInfo = info
		cache[info.Name] = detail
		out = append(out, detail)
	}
	c.details = cache
	slices.SortFunc(out, func(a, b VMDetail) int { return cmp.Compare(a.Name, b.Name) })
	return out, failed, errs.Err()
}

// getLocked reads one VM's detail from the wrapped Provider. c.mu must be held.
func (c *CachingProvider) getLocked(ctx context.Context, vmName string) (VMDetail, error) {
	detail, err := c.Provider.Get(ctx, vmName)
	if err != nil {
		return VMDetail{}, err
	}
	return c.withDefaults(detail), nil
}

// withDefaults fills in any resource the VM does not report for itself from
// the provider's defaults, see Provider.DefaultResources. A VM that has not
// been started yet commonly reports none, and showing nothing at all is of
// less use than showing what it will be created with.
func (c *CachingProvider) withDefaults(detail VMDetail) VMDetail {
	defaults := c.DefaultResources()
	if detail.NumCores == 0 {
		detail.NumCores = defaults.NumCores
	}
	if detail.MemGiB == 0 {
		detail.MemGiB = defaults.MemGiB
	}
	if detail.DiskGiB == 0 {
		detail.DiskGiB = defaults.DiskGiB
	}
	return detail
}

// Get implements Provider, returning the full details of a single VM from the
// cache if they are held there and reading them from the wrapped Provider if
// they are not. Missing resources are filled in from the provider's defaults,
// see Provider.DefaultResources, and nothing is cached for a VM that cannot be
// read.
//
// A cached VM is served entirely from the cache, its VMInfo included, which
// may be as old as the entry: use GetAll where the state has to be current.
// Nothing is dropped from the cache here either, since one VM says nothing
// about which others still exist; it is GetAll that prunes it.
func (c *CachingProvider) Get(ctx context.Context, vmName string) (VMDetail, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d, ok := c.details[vmName]; ok {
		return d, nil
	}
	detail, err := c.getLocked(ctx, vmName)
	if err != nil {
		return VMDetail{}, err
	}
	if c.details == nil {
		c.details = make(map[string]VMDetail)
	}
	c.details[vmName] = detail
	return detail, nil
}
