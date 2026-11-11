// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/sync/synctestutil"
	"cloudeng.io/vms"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// startedPool returns a started pool of one VM, backed by a mock, for the
// tests of a set of them.
func startedPool(ctx context.Context, t *testing.T, name string, opts ...vmspool.Option) *vmspool.Pool {
	t.Helper()
	p := vmspool.New(vmstestutil.NewMockFactory(true), name,
		append([]vmspool.Option{
			vmspool.WithSize(1),
			vmspool.WithCreateBackoff(fastCreateBackoff()),
		}, opts...)...)
	if err := p.Start(ctx); err != nil {
		t.Fatalf("%v: Start: %v", name, err)
	}
	return p
}

// TestSetNaming covers what a set is for: addressing a pool by name, and
// saying so when the name is not one of them.
func TestSetNaming(t *testing.T) {
	ctx := context.Background()
	one, two := startedPool(ctx, t, "one"), startedPool(ctx, t, "two")
	set := vmspool.NewSet(nil, nil, two, one)
	defer set.Close(ctx) //nolint:errcheck // the test has already reported anything that matters.

	if got, want := set.Len(), 2; got != want {
		t.Errorf("Len: got %v, want %v", got, want)
	}
	// Sorted, whatever order the set were given in, so that output listing
	// them is stable.
	if got, want := set.Names(), []string{"one", "two"}; !slices.Equal(got, want) {
		t.Errorf("Names: got %v, want %v", got, want)
	}
	got, err := set.Pool("one")
	if err != nil {
		t.Fatalf("Pool: %v", err)
	}
	if got != one {
		t.Error("Pool returned a pool other than the one of that name")
	}
	all := set.All()
	if names := slices.Sorted(maps.Keys(all)); !slices.Equal(names, []string{"one", "two"}) {
		t.Errorf("All: got %v, want one and two", names)
	}
	if all["two"] != two {
		t.Error("All returned a pool other than the one of that name")
	}

	// An unknown name is reported, by name, by everything that takes one.
	_, err = set.Pool("three")
	if !errors.Is(err, vmspool.ErrNoSuchPool) {
		t.Errorf("Pool: got %v, want ErrNoSuchPool", err)
	}
	if err == nil || !strings.Contains(err.Error(), "three") {
		t.Errorf("the error does not name the pool: %v", err)
	}
	if _, err := set.Acquire(ctx, "three"); !errors.Is(err, vmspool.ErrNoSuchPool) {
		t.Errorf("Acquire: got %v, want ErrNoSuchPool", err)
	}
	if _, err := set.Refresh(ctx, "three"); !errors.Is(err, vmspool.ErrNoSuchPool) {
		t.Errorf("Refresh: got %v, want ErrNoSuchPool", err)
	}
}

// TestSetAcquireAndRefresh covers the two operations a set passes on to one
// of its set, which is all they are: what they do to the pool is the pool's
// own business, tested there.
func TestSetAcquireAndRefresh(t *testing.T) {
	ctx := context.Background()
	set := vmspool.NewSet(nil, nil, startedPool(ctx, t, "one"))
	defer set.Close(ctx) //nolint:errcheck // the test has already reported anything that matters.

	vm, err := set.Acquire(ctx, "one")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if got, want := vm.PoolName(), "one"; got != want {
		t.Errorf("the VM came from pool %v, want %v", got, want)
	}
	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
	// The pool replaces the VM that was taken, so there is something to
	// refresh again; Refresh reports what it replaced, 0 included.
	if _, err := set.Refresh(ctx, "one"); err != nil {
		t.Errorf("Refresh: %v", err)
	}
}

// TestSetClosed covers a set that has been closed, which is what an
// application shutting down leaves behind whilst work is still running against
// it: nothing is handed out, and what the set is is still reported.
func TestSetClosed(t *testing.T) {
	ctx := context.Background()
	set := vmspool.NewSet(nil, nil, startedPool(ctx, t, "one"))
	if err := set.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, tc := range []struct {
		op   string
		err  error
		call func() error
	}{
		{"Pool", nil, func() error { _, err := set.Pool("one"); return err }},
		{"Acquire", nil, func() error { _, err := set.Acquire(ctx, "one"); return err }},
		{"Refresh", nil, func() error { _, err := set.Refresh(ctx, "one"); return err }},
	} {
		err := tc.call()
		if !errors.Is(err, vmspool.ErrPoolsClosed) {
			t.Errorf("%v: got %v, want ErrPoolsClosed", tc.op, err)
		}
	}
	if !strings.Contains(setCloseErr(set, "one"), "one") {
		t.Error("the error does not name the pool")
	}

	// A closed set still says what it holds, so that a last report of the
	// set is possible, and no longer has anything to hand out.
	if got, want := set.Len(), 1; got != want {
		t.Errorf("Len: got %v, want %v", got, want)
	}
	if got, want := set.Names(), []string{"one"}; !slices.Equal(got, want) {
		t.Errorf("Names: got %v, want %v", got, want)
	}
	if len(set.All()) != 1 {
		t.Errorf("All: got %v set, want 1", len(set.All()))
	}
	if available, pending := set.AvailableAndPending(); available || pending {
		t.Errorf("a closed set reports available %v, pending %v", available, pending)
	}

	// Closing twice returns the result of the first close rather than an error
	// of its own, as closing a pool or a queue twice does.
	if err := set.Close(ctx); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

// setCloseErr returns the text of the error a closed set reports for the
// named pool.
func setCloseErr(set *vmspool.Set, name string) string {
	_, err := set.Pool(name)
	if err == nil {
		return ""
	}
	return err.Error()
}

// availabilityWithin polls until the set reports an availability that want
// accepts, which is not immediate: a pool fills itself in the background. desc
// says what was being waited for, for the failure.
func availabilityWithin(t *testing.T, set *vmspool.Set, want func(available, pending bool) bool, desc string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		available, pending := set.AvailableAndPending()
		if want(available, pending) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %v: available %v, pending %v", desc, available, pending)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestSetAvailableAndPending covers what a set reports about its pools without
// being asked about a particular one: whether any of them can hand out a VM
// now, and whether any has one on the way.
func TestSetAvailableAndPending(t *testing.T) {
	ctx := context.Background()
	set := vmspool.NewSet(nil, nil, startedPool(ctx, t, "one"))
	defer set.Close(ctx) //nolint:errcheck // the test has already reported anything that matters.

	// Start has waited for the first VM, so there is one to be had.
	if available, _ := set.AvailableAndPending(); !available {
		t.Error("a filled set reports nothing available")
	}
	vm, err := set.Acquire(ctx, "one")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer vm.Delete(context.Background()) //nolint:errcheck // best effort cleanup.
	// The pool holds one VM and it is now held by a caller, so the set has
	// nothing to hand out and nothing on the way either: the replacement is
	// not created until the caller releases it.
	availabilityWithin(t, set, func(available, pending bool) bool { return !available && !pending },
		"a fully acquired set to report neither")

	// An empty set has neither, rather than panicking on a set with no pools
	// at all.
	if available, pending := vmspool.NewSet(nil, nil).AvailableAndPending(); available || pending {
		t.Errorf("a set with no pools reports available %v, pending %v", available, pending)
	}
}

// blockingProvider creates VMs through a mock, holding each one up part way
// through its creation once blocking is set: the VM exists, and so is pending,
// but is not ready to be acquired, see Pool.Stats.
type blockingProvider struct {
	*vmstestutil.MockFactory
	blocking atomic.Bool
	release  chan struct{}
}

func (p *blockingProvider) New(ctx context.Context) (vms.Instance, error) {
	inst, err := p.MockFactory.New(ctx)
	if err != nil || !p.blocking.Load() {
		return inst, err
	}
	return blockingInstance{Instance: inst, release: p.release}, nil
}

type blockingInstance struct {
	vms.Instance
	release chan struct{}
}

func (i blockingInstance) Clone(ctx context.Context) error {
	select {
	case <-i.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	return i.Instance.Clone(ctx)
}

// TestSetAvailableAndPendingTogether covers a set that is both at once, which
// is what keeping the two answers apart is for: a caller gating on capacity
// counts the pending VM, one waiting for the set to be full does not.
func TestSetAvailableAndPendingTogether(t *testing.T) {
	ctx := context.Background()
	provider := &blockingProvider{
		MockFactory: vmstestutil.NewMockFactory(true),
		release:     make(chan struct{}),
	}
	pool := vmspool.New(provider, "two",
		vmspool.WithSize(2), vmspool.WithCreateBackoff(fastCreateBackoff()))
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	set := vmspool.NewSet(nil, nil, pool)
	// Start waits for the first VM only, so wait for the pool to be full
	// before holding anything up.
	deadline := time.Now().Add(5 * time.Second)
	for pool.Stats().Available < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for the pool to fill: %v", pool.Stats())
		}
		time.Sleep(time.Millisecond)
	}

	// Deleting an acquired VM has the pool replace it, and that replacement is
	// held up in its creation, so the set has one VM to hand out and one on
	// the way.
	provider.blocking.Store(true)
	vm, err := set.Acquire(ctx, "two")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := vm.Delete(ctx); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	availabilityWithin(t, set, func(available, pending bool) bool { return available && pending },
		"a replenishing set to report a VM available and another pending")

	close(provider.release)
	if err := set.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestSetSubscribe covers the set's change signal, which is its tracker's, so
// that a reader watches the set rather than reaching for what counts for it.
func TestSetSubscribe(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, 5*time.Second)()
	// Cancelled before the leak assertion runs, see
	// TestSetCloseSharedComponents.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := vmspool.NewStatusTracker(ctx, 10)
	set := vmspool.NewSet(nil, tracker, startedPool(ctx, t, "one", tracker.Track(ctx)))
	defer func() { _ = set.Close(ctx) }()

	ch, release := set.Subscribe(ctx)
	defer release()
	// Acquiring a VM has the pool replace it, which is counted and so
	// signalled.
	if _, err := set.Acquire(ctx, "one"); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("the set reported no change after a VM was acquired")
	}
	// Releasing twice is harmless.
	release()
}

// TestSetCloseSharedComponents covers the set closing what its set share:
// the queue the finished VMs are parked in and the tracker counting their
// events, both of which outlive any one pool and so can be closed by nothing
// else.
func TestSetCloseSharedComponents(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, 5*time.Second)()
	// A cancellable context, cancelled before the leak assertion runs, because
	// a started mock VM leaves a goroutine waiting on it, see
	// vmstestutil.Mock.Start.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const outcome = "done"
	cq := vmspool.NewCompletionQueue(ctx, vmspool.WithCompletionOutcome(outcome, 2, time.Hour))
	tracker := vmspool.NewStatusTracker(ctx, 10)
	pool := startedPool(ctx, t, "one", vmspool.WithCompletionQueue(cq), tracker.Track(ctx))
	set := vmspool.NewSet(cq, tracker, pool)

	if got := set.CompletionQueue(); got != cq {
		t.Error("CompletionQueue returned a queue other than the set's")
	}
	if got := set.StatusTracker(); got != tracker {
		t.Error("StatusTracker returned a tracker other than the set's")
	}
	// The pool has filled, so the tracker has counted it: the set's pools
	// report to the set's tracker.
	countersWithin(t, tracker, "one", vmspool.EventVMCreated, 1)

	// A VM is parked in the queue, which the set releases when it is closed
	// rather than leaving it running with the VM still in it.
	vm, err := set.Acquire(ctx, "one")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	cq.Push(vmspool.CompletionEvent{VM: completed{vm: vm}, Outcome: outcome})
	if got, want := cq.NumEventsPerPool("one"), 1; got != want {
		t.Fatalf("the queue holds %v of the pool's VMs, want %v", got, want)
	}

	if err := set.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}
	if got := cq.NumEvents(); got != 0 {
		t.Errorf("the queue still holds %v VMs after the set was closed", got)
	}
	// The queue and the tracker are closed with the set: a closed queue hands
	// out no channel, and a closed tracker keeps the counters it had.
	if cq.Out(outcome) != nil {
		t.Error("the completion queue was left open")
	}
	if counters, _ := tracker.Counters("one"); len(counters) == 0 {
		t.Error("the tracker lost the counters it had accumulated")
	}
}

// TestSetWithoutSharedComponents covers a set created with neither a queue
// nor a tracker, which is an application that parks no finished work and
// counts nothing: closing it must not reach for either.
func TestSetWithoutSharedComponents(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, 5*time.Second)()
	// Cancelled before the leak assertion runs, see
	// TestSetCloseSharedComponents.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	set := vmspool.NewSet(nil, nil, startedPool(ctx, t, "one"))
	if got := set.CompletionQueue(); got != nil {
		t.Error("a set created without a queue reports one")
	}
	if got := set.StatusTracker(); got != nil {
		t.Error("a set created without a tracker reports one")
	}
	// Nothing is counting, so there is no change signal rather than a channel
	// that is never signalled.
	ch, release := set.Subscribe(ctx)
	if ch != nil {
		t.Error("a set created without a tracker reports a change signal")
	}
	release()
	if err := set.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// listingProvider is a Provider that lists the VMs it was given and reports
// deleting them, which is all ListVMs and DeleteVMs ask of one.
type listingProvider struct {
	name     string
	vms      []string
	listErr  error
	delErr   error
	deleteMu sync.Mutex
	timeout  time.Duration // the stopTimeout Delete was called with
}

func (p *listingProvider) Name() string { return p.name }

func (p *listingProvider) DefaultResources() vmspool.VMResources {
	return vmspool.VMResources{}
}

func (p *listingProvider) Details() vmspool.ProviderDetails { return nil }

func (p *listingProvider) New(context.Context) (vms.Instance, error) {
	return nil, errors.New("this provider creates no VMs")
}

func (p *listingProvider) List(context.Context) ([]vmspool.VMInfo, error) {
	if p.listErr != nil {
		return nil, p.listErr
	}
	out := make([]vmspool.VMInfo, 0, len(p.vms))
	for _, name := range p.vms {
		out = append(out, vmspool.VMInfo{Name: name, Pool: p.name})
	}
	return out, nil
}

func (p *listingProvider) Get(context.Context, string) (vmspool.VMDetail, error) {
	return vmspool.VMDetail{}, errors.New("this provider inspects no VMs")
}

func (p *listingProvider) Delete(_ context.Context, stopTimeout time.Duration) ([]string, error) {
	p.deleteMu.Lock()
	p.timeout = stopTimeout
	p.deleteMu.Unlock()
	if p.delErr != nil {
		return nil, p.delErr
	}
	return slices.Clone(p.vms), nil
}

func (p *listingProvider) stopTimeout() time.Duration {
	p.deleteMu.Lock()
	defer p.deleteMu.Unlock()
	return p.timeout
}

// providerNew builds one provider, or fails to, as a ProviderNew does.
type providerNew struct {
	provider *listingProvider
	err      error
}

func (pn providerNew) Name() string {
	if pn.provider == nil {
		return "unbuildable"
	}
	return pn.provider.name
}

func (pn providerNew) New(context.Context) (vmspool.Provider, error) {
	if pn.err != nil {
		return nil, pn.err
	}
	return pn.provider, nil
}

// vmNames is the names of the listed VMs, in the order they were returned.
func vmNames(infos []vmspool.VMInfo) []string {
	out := make([]string, 0, len(infos))
	for _, info := range infos {
		out = append(out, info.Name)
	}
	return out
}

// TestListVMs covers enumerating the VMs of several backends at once, which is
// what reporting on the VMs left behind by a previous run needs: it builds
// each provider itself and needs no pool.
func TestListVMs(t *testing.T) {
	ctx := context.Background()
	one := &listingProvider{name: "one", vms: []string{"one-a", "one-b"}}
	two := &listingProvider{name: "two", vms: []string{"two-a"}}

	infos, err := vmspool.ListVMs(ctx, providerNew{provider: one}, providerNew{provider: two})
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if got, want := vmNames(infos), []string{"one-a", "one-b", "two-a"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Each VM says which pool it came from, which is how a sweep tells the VMs
	// of one pool from another's.
	for _, info := range infos {
		if !strings.HasPrefix(info.Name, info.Pool) {
			t.Errorf("%v: reported pool %v", info.Name, info.Pool)
		}
	}
	// Nothing to list from nothing.
	if infos, err := vmspool.ListVMs(ctx); err != nil || len(infos) != 0 {
		t.Errorf("no providers: got %v, %v", infos, err)
	}
}

// TestListVMsFailures covers the failures accumulating rather than one backend
// hiding what the others report: one that cannot be built and one that cannot
// list.
func TestListVMsFailures(t *testing.T) {
	ctx := context.Background()
	good := &listingProvider{name: "good", vms: []string{"good-a"}}
	unbuildable := errors.New("this backend cannot be built")
	unlistable := errors.New("this backend cannot list its VMs")

	infos, err := vmspool.ListVMs(ctx,
		providerNew{provider: good},
		providerNew{err: unbuildable},
		providerNew{provider: &listingProvider{name: "flaky", listErr: unlistable}})
	if !errors.Is(err, unbuildable) || !errors.Is(err, unlistable) {
		t.Errorf("the errors of both failing backends were not reported: %v", err)
	}
	// What the backend that worked reports is still returned.
	if got, want := vmNames(infos), []string{"good-a"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDeleteVMs covers sweeping the VMs of several backends at once, which is
// how a run cleans up after one that exited without doing so itself.
func TestDeleteVMs(t *testing.T) {
	ctx := context.Background()
	one := &listingProvider{name: "one", vms: []string{"one-a", "one-b"}}
	two := &listingProvider{name: "two", vms: []string{"two-a"}}

	deleted, err := vmspool.DeleteVMs(ctx, time.Minute, providerNew{provider: one}, providerNew{provider: two})
	if err != nil {
		t.Fatalf("DeleteVMs: %v", err)
	}
	if got, want := deleted, []string{"one-a", "one-b", "two-a"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// Each backend is given the timeout it is to stop a running VM within.
	for _, p := range []*listingProvider{one, two} {
		if got, want := p.stopTimeout(), time.Minute; got != want {
			t.Errorf("%v: stop timeout %v, want %v", p.name, got, want)
		}
	}
}

// TestDeleteVMsFailures covers a backend that cannot sweep its VMs: it is
// reported and the others are swept regardless, since leaving VMs behind
// because another backend is broken is the worse outcome.
func TestDeleteVMsFailures(t *testing.T) {
	ctx := context.Background()
	good := &listingProvider{name: "good", vms: []string{"good-a"}}
	undeletable := errors.New("this backend cannot delete its VMs")

	deleted, err := vmspool.DeleteVMs(ctx, time.Second,
		providerNew{provider: good},
		providerNew{provider: &listingProvider{name: "flaky", vms: []string{"flaky-a"}, delErr: undeletable}})
	if !errors.Is(err, undeletable) {
		t.Errorf("got %v, want the backend's own error", err)
	}
	if got, want := deleted, []string{"good-a"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
