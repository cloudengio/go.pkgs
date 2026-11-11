// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/sync/errgroup"
)

// Set is a set of named pools that are held and closed together, along with
// the completion queue and the status tracker they share.
type Set struct {
	// Fixed when the set is created, so read without the mutex.
	pools      map[string]*Pool
	names      []string
	tracker    *StatusTracker
	completion *CompletionQueue

	mu     sync.Mutex
	closed bool
	// closeOnce makes Close idempotent, closeErr is what it reports to every
	// caller.
	closeOnce sync.Once
	closeErr  error
}

// NewSet returns a set holding pools, keyed by the name each was created
// with, and already started, see Pool.Start. cq and tracker are what the pools
// share and either may be nil; the set owns both and closes them after the
// pools, see Close. Two pools of one name are one pool, the last winning.
func NewSet(cq *CompletionQueue, tracker *StatusTracker, pools ...*Pool) *Set {
	s := &Set{
		pools:      make(map[string]*Pool, len(pools)),
		tracker:    tracker,
		completion: cq,
	}
	for _, pool := range pools {
		s.pools[pool.Name()] = pool
	}
	s.names = slices.Sorted(maps.Keys(s.pools))
	return s
}

var (
	// ErrNoSuchPool is returned for a name the set holds no pool for.
	ErrNoSuchPool = errors.New("no such pool")
	// ErrPoolsClosed is returned by the operations of a closed set.
	ErrPoolsClosed = errors.New("pools is closed")
)

// poolError annotates err with the name of the pool it concerns.
func poolError(name string, err error) error {
	return fmt.Errorf("vm_pool %s: %w", name, err)
}

func (s *Set) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Pool returns the named pool, ErrNoSuchPool if the set holds none of that
// name, or ErrPoolsClosed if the set has been closed.
func (s *Set) Pool(name string) (*Pool, error) {
	if s.isClosed() {
		return nil, poolError(name, ErrPoolsClosed)
	}
	pool, ok := s.pools[name]
	if !ok {
		return nil, poolError(name, ErrNoSuchPool)
	}
	return pool, nil
}

// All returns every pool, keyed by name, as a map of the caller's own. A
// closed set returns its pools too: a closed pool still reports what it is.
func (s *Set) All() map[string]*Pool {
	return maps.Clone(s.pools)
}

// Len returns the number of pools in the set.
func (s *Set) Len() int {
	return len(s.pools)
}

// Names returns the names of the pools, sorted, so that listings are stable.
func (s *Set) Names() []string {
	return slices.Clone(s.names)
}

// CompletionQueue returns the queue the pools' finished VMs are parked in, or
// nil if the set has none. The set owns it: fill and drain it, do not close it.
func (s *Set) CompletionQueue() *CompletionQueue {
	return s.completion
}

// StatusTracker returns the tracker counting the pools' events, or nil if the
// set has none. The set owns it.
func (s *Set) StatusTracker() *StatusTracker {
	return s.tracker
}

// Subscribe returns a channel signalled whenever any pool's counters change
// and a function that must be called to release the subscription, see
// StatusTracker.Subscribe.
//
// A set with no tracker has no signal: the channel is nil, which blocks
// forever rather than reporting changes that are not being counted, and
// releasing it does nothing.
func (s *Set) Subscribe(ctx context.Context) (<-chan struct{}, func()) {
	if s.tracker == nil {
		return nil, func() {}
	}
	return s.tracker.Subscribe(ctx)
}

// Acquire returns a VM from the named pool, see Pool.Acquire.
func (s *Set) Acquire(ctx context.Context, poolName string) (*VM, error) {
	pool, err := s.Pool(poolName)
	if err != nil {
		return nil, err
	}
	return pool.Acquire(ctx)
}

// Refresh replaces the VMs the named pool has available and returns how many
// were replaced, see Pool.Refresh.
func (s *Set) Refresh(ctx context.Context, poolName string) (int, error) {
	pool, err := s.Pool(poolName)
	if err != nil {
		return 0, err
	}
	return pool.Refresh(ctx), nil
}

// Close closes every pool concurrently, then the completion queue and the
// status tracker, accumulating their errors. The set is out of service from
// the moment it is called.
//
// The queue goes after the pools, since closing it deletes the VMs it holds,
// and the tracker last, since a pool reports to it until it has stopped. It is
// idempotent: a later call returns the first one's result.
func (s *Set) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.closeOnce.Do(func() {
		var g errgroup.T
		for _, pool := range s.pools {
			g.Go(func() error {
				return pool.Close(ctx)
			})
		}
		var errs errors.M
		errs.Append(g.Wait())
		if s.completion != nil {
			errs.Append(s.completion.Close(ctx))
		}
		if s.tracker != nil {
			s.tracker.Close()
		}
		s.closeErr = errs.Err()
	})
	return s.closeErr
}

// AvailableAndPending reports whether any pool has a VM that Acquire can
// return without waiting for one to be created, and whether any pool has one
// being created and so acquirable shortly, see Pool.Stats. The two are
// independent: a set filling itself is both. A closed set is neither.
func (s *Set) AvailableAndPending() (available, pending bool) {
	if s.isClosed() {
		return false, false
	}
	for _, pool := range s.pools {
		stats := pool.Stats()
		if stats.Available > 0 {
			available = true
			if pending {
				return
			}
		}
		if stats.Pending > 0 {
			pending = true
			if available {
				return
			}
		}
	}
	return available, pending
}

// ListVMs lists all VMs from the given providers, sorted by VM name. It returns
// a slice of VMInfo and any error encountered. Errors are accumulated from all
// providers.
// Note, that this function is intentionally independent of any pool and
// operates directly on the providers.
func ListVMs(ctx context.Context, providerNew ...ProviderNew) ([]VMInfo, error) {
	var g errgroup.T
	var vms []VMInfo
	var mu sync.Mutex
	for _, pn := range providerNew {
		g.Go(func() error {
			p, err := pn.New(ctx)
			if err != nil {
				return err
			}
			poolVMs, err := p.List(ctx)
			if err != nil {
				return err
			}
			mu.Lock()
			vms = append(vms, poolVMs...)
			mu.Unlock()
			return nil
		})
	}
	err := g.Wait()
	slices.SortFunc(vms, func(a, b VMInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return vms, err
}

// DeleteVMs deletes all VMs from the given providers, sorted by VM name. It
// returns a slice of deleted VM IDs and any error encountered. Errors are
// accumulated from all providers.
// Note, that this function is intentionally independent of any pool and
// operates directly on the providers.
func DeleteVMs(ctx context.Context, stopTimeout time.Duration, providerNew ...ProviderNew) ([]string, error) {
	var g errgroup.T
	var vms []string
	var mu sync.Mutex
	for _, pn := range providerNew {
		g.Go(func() error {
			p, err := pn.New(ctx)
			if err != nil {
				return err
			}
			deleted, err := p.Delete(ctx, stopTimeout)
			if err != nil {
				return err
			}
			mu.Lock()
			vms = append(vms, deleted...)
			mu.Unlock()
			return nil
		})
	}
	err := g.Wait()
	slices.Sort(vms)
	return vms, err
}
