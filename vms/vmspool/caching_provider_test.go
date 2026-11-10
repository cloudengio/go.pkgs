// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"cloudeng.io/vms"
	"cloudeng.io/vms/ociimage"
	"cloudeng.io/vms/vmspool"
)

// countingProvider is a Provider whose listing and per-VM details are set by
// the test, and which counts the Get calls that reach it, so that a cache hit
// can be told from a read.
type countingProvider struct {
	mu       sync.Mutex
	infos    []vmspool.VMInfo
	details  map[string]vmspool.VMDetail
	failures map[string]error
	listErr  error
	gets     map[string]int
}

func newCountingProvider() *countingProvider {
	return &countingProvider{
		details:  map[string]vmspool.VMDetail{},
		failures: map[string]error{},
		gets:     map[string]int{},
	}
}

// vmInfo is the listing entry for a VM, which is the part that changes as it
// runs.
func vmInfo(name, state string, running bool) vmspool.VMInfo {
	return vmspool.VMInfo{Name: name, State: state, Running: running}
}

// add gives the provider a VM with the listing entry info and the resources
// cores/mem, which is what Get reports for it.
func (p *countingProvider) add(info vmspool.VMInfo, cores, mem int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.infos = append(p.infos, info)
	p.details[info.Name] = vmspool.VMDetail{
		VMInfo:      info,
		VMResources: vmspool.VMResources{NumCores: cores, MemGiB: mem},
	}
}

// setListing replaces the listing, standing for VMs that have come or gone.
func (p *countingProvider) setListing(infos ...vmspool.VMInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.infos = infos
	for _, info := range infos {
		d := p.details[info.Name]
		d.VMInfo = info
		p.details[info.Name] = d
	}
}

func (p *countingProvider) fail(name string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures[name] = err
}

func (p *countingProvider) getCount(name string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gets[name]
}

func (p *countingProvider) Name() string { return "counting" }

func (p *countingProvider) DefaultResources() vmspool.VMResources {
	return vmspool.VMResources{NumCores: 1, MemGiB: 2, DiskGiB: 3}
}

func (p *countingProvider) Details() vmspool.ProviderDetails { return countingDetails{} }

func (p *countingProvider) New(context.Context) (vms.Instance, error) {
	return nil, errors.New("countingProvider creates no VMs")
}

func (p *countingProvider) List(context.Context) ([]vmspool.VMInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listErr != nil {
		return nil, p.listErr
	}
	return append([]vmspool.VMInfo(nil), p.infos...), nil
}

func (p *countingProvider) Get(_ context.Context, vmName string) (vmspool.VMDetail, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gets[vmName]++
	if err := p.failures[vmName]; err != nil {
		return vmspool.VMDetail{}, err
	}
	d, ok := p.details[vmName]
	if !ok {
		return vmspool.VMDetail{}, errors.New("no such VM: " + vmName)
	}
	return d, nil
}

func (p *countingProvider) Delete(context.Context, time.Duration) ([]string, error) {
	return nil, nil
}

type countingDetails struct{}

func (countingDetails) Image() ociimage.Reference {
	return ociimage.Reference{Name: "counting-image"}
}

// byName indexes details by VM name, for a test asserting on one of them.
func byName(t *testing.T, details []vmspool.VMDetail) map[string]vmspool.VMDetail {
	t.Helper()
	out := make(map[string]vmspool.VMDetail, len(details))
	for _, d := range details {
		out[d.Name] = d
	}
	return out
}

// TestCachingProviderCaches covers what the cache is for: the details of a VM
// are read once, however many times they are reported, and in name order
// whatever order the provider lists them in.
func TestCachingProviderCaches(t *testing.T) {
	ctx := context.Background()
	p := newCountingProvider()
	p.add(vmInfo("vm-b", "running", true), 8, 16)
	p.add(vmInfo("vm-a", "stopped", false), 4, 8)
	c := &vmspool.CachingProvider{Provider: p}

	for i := range 3 {
		got, failed, err := c.GetAll(ctx)
		if err != nil {
			t.Fatalf("GetAll: %v", err)
		}
		if len(failed) != 0 {
			t.Errorf("failed: got %v, want none", failed)
		}
		if len(got) != 2 {
			t.Fatalf("got %v VMs, want 2", len(got))
		}
		if got[0].Name != "vm-a" || got[1].Name != "vm-b" {
			t.Errorf("call %v: got %q, %q, want them ordered by name", i, got[0].Name, got[1].Name)
		}
		if got, want := got[1].NumCores, 8; got != want {
			t.Errorf("cores: got %v, want %v", got, want)
		}
	}
	// Three reports, one read of each VM.
	for _, name := range []string{"vm-a", "vm-b"} {
		if got := p.getCount(name); got != 1 {
			t.Errorf("%v was read %v times, want once", name, got)
		}
	}

	// Get serves the same cache, so it reads nothing either.
	if _, err := c.Get(ctx, "vm-a"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := p.getCount("vm-a"); got != 1 {
		t.Errorf("vm-a was read %v times, want once", got)
	}
}

// TestCachingProviderStateIsNotCached covers the one part of a VMDetail that
// must not be served from the cache: a VM that was read when it was stopped
// and is now running is reported as running.
func TestCachingProviderStateIsNotCached(t *testing.T) {
	ctx := context.Background()
	p := newCountingProvider()
	p.add(vmInfo("vm-a", "stopped", false), 4, 8)
	c := &vmspool.CachingProvider{Provider: p}

	if _, _, err := c.GetAll(ctx); err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	p.setListing(vmInfo("vm-a", "running", true))

	got, _, err := c.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %v VMs, want 1", len(got))
	}
	if !got[0].Running || got[0].State != "running" {
		t.Errorf("the cached VM reports %q/running=%v, want the state the listing gives it",
			got[0].State, got[0].Running)
	}
	// Its resources still came from the cache.
	if got, want := got[0].NumCores, 4; got != want {
		t.Errorf("cores: got %v, want %v", got, want)
	}
	if got := p.getCount("vm-a"); got != 1 {
		t.Errorf("vm-a was read %v times, want once", got)
	}
}

// TestCachingProviderEvicts covers the cache holding the VMs that exist: one
// that has gone is dropped, and one created in its place is read.
func TestCachingProviderEvicts(t *testing.T) {
	ctx := context.Background()
	p := newCountingProvider()
	p.add(vmInfo("vm-a", "running", true), 4, 8)
	c := &vmspool.CachingProvider{Provider: p}

	if _, _, err := c.GetAll(ctx); err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	// vm-a is deleted and vm-b created, as a pool replaces an acquired VM.
	p.add(vmInfo("vm-b", "running", true), 2, 4)
	p.setListing(vmInfo("vm-b", "running", true))

	got, _, err := c.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(got) != 1 || got[0].Name != "vm-b" {
		t.Fatalf("got %+v, want vm-b alone", got)
	}

	// vm-a comes back under the same name, so it is read again rather than
	// reported with the resources the VM that has gone had.
	p.add(vmInfo("vm-a", "stopped", false), 16, 32)
	p.setListing(vmInfo("vm-a", "stopped", false), vmInfo("vm-b", "running", true))
	got, _, err = c.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if got, want := byName(t, got)["vm-a"].NumCores, 16; got != want {
		t.Errorf("cores: got %v, want %v", got, want)
	}
	if got := p.getCount("vm-a"); got != 2 {
		t.Errorf("vm-a was read %v times, want twice", got)
	}
}

// TestCachingProviderGetFails covers a VM that exists but cannot be
// inspected: it is still reported, with the provider's defaults, and it is not
// cached, so a failure that passes is not reported for as long as the VM
// exists.
func TestCachingProviderGetFails(t *testing.T) {
	ctx := context.Background()
	errBusy := errors.New("busy")
	p := newCountingProvider()
	p.add(vmInfo("vm-a", "running", true), 4, 8)
	p.add(vmInfo("vm-b", "running", true), 0, 0)
	p.fail("vm-b", errBusy)
	c := &vmspool.CachingProvider{Provider: p}

	got, failed, err := c.GetAll(ctx)
	if !errors.Is(err, errBusy) {
		t.Errorf("GetAll: got %v, want it to report the failure", err)
	}
	if len(failed) != 1 || failed[0] != "vm-b" {
		t.Errorf("failed: got %v, want [vm-b]", failed)
	}
	// Both VMs are reported, the failing one with the listing and the
	// provider's defaults.
	if len(got) != 2 {
		t.Fatalf("got %v VMs, want both of them", len(got))
	}
	b := byName(t, got)["vm-b"]
	if !b.Running || b.State != "running" {
		t.Errorf("vm-b: got %+v, want what the listing says", b.VMInfo)
	}
	defaults := p.DefaultResources()
	if b.VMResources != defaults {
		t.Errorf("vm-b resources: got %+v, want the defaults %+v", b.VMResources, defaults)
	}

	// The failure was not cached: once it passes, the real resources are
	// reported.
	p.fail("vm-b", nil)
	p.add(vmInfo("vm-b", "running", true), 6, 12)
	p.setListing(vmInfo("vm-a", "running", true), vmInfo("vm-b", "running", true))
	got, failed, err = c.GetAll(ctx)
	if err != nil || len(failed) != 0 {
		t.Fatalf("GetAll: got %v, %v, want no failures", failed, err)
	}
	if got, want := byName(t, got)["vm-b"].NumCores, 6; got != want {
		t.Errorf("cores: got %v, want %v", got, want)
	}

	// Get reports the failure rather than defaults, since a caller asking for
	// one VM has nothing to report it alongside.
	p.fail("vm-a", errBusy)
	c2 := &vmspool.CachingProvider{Provider: p}
	if _, err := c2.Get(ctx, "vm-a"); !errors.Is(err, errBusy) {
		t.Errorf("Get: got %v, want %v", err, errBusy)
	}
}

// TestCachingProviderListFails covers a listing that cannot be had at all,
// which leaves nothing to report and no reason to trust the cache.
func TestCachingProviderListFails(t *testing.T) {
	ctx := context.Background()
	errDown := errors.New("daemon is not running")
	p := newCountingProvider()
	p.add(vmInfo("vm-a", "running", true), 4, 8)
	c := &vmspool.CachingProvider{Provider: p}

	p.listErr = errDown
	got, failed, err := c.GetAll(ctx)
	if !errors.Is(err, errDown) {
		t.Errorf("GetAll: got %v, want %v", err, errDown)
	}
	if got != nil || failed != nil {
		t.Errorf("got %v, %v, want nothing reported", got, failed)
	}
}

// TestCachingProviderZeroValue covers reaching Get before GetAll, which is
// the first thing a caller holding one of these does if it reports on a
// single VM.
func TestCachingProviderZeroValue(t *testing.T) {
	ctx := context.Background()
	p := newCountingProvider()
	p.add(vmInfo("vm-a", "running", true), 4, 8)
	c := &vmspool.CachingProvider{Provider: p}

	got, err := c.Get(ctx, "vm-a")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got, want := got.NumCores, 4; got != want {
		t.Errorf("cores: got %v, want %v", got, want)
	}
	// It was cached, so a second call reads nothing.
	if _, err := c.Get(ctx, "vm-a"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := p.getCount("vm-a"); got != 1 {
		t.Errorf("vm-a was read %v times, want once", got)
	}

	// The wrapped provider is reached for everything the cache does not hold.
	if got, want := c.Name(), "counting"; got != want {
		t.Errorf("Name: got %v, want %v", got, want)
	}
	if got, want := c.Details().Image().Name, "counting-image"; got != want {
		t.Errorf("Image: got %v, want %v", got, want)
	}
}

// TestCachingProviderConcurrent covers concurrent reporting, which is what an
// application serving a status page from several requests does.
func TestCachingProviderConcurrent(t *testing.T) {
	ctx := context.Background()
	p := newCountingProvider()
	for _, n := range []string{"vm-a", "vm-b", "vm-c"} {
		p.add(vmInfo(n, "running", true), 4, 8)
	}
	c := &vmspool.CachingProvider{Provider: p}

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, _, err := c.GetAll(ctx); err != nil {
				t.Errorf("GetAll: %v", err)
			}
			if _, err := c.Get(ctx, "vm-a"); err != nil {
				t.Errorf("Get: %v", err)
			}
		})
	}
	wg.Wait()
	for _, name := range []string{"vm-a", "vm-b", "vm-c"} {
		if got := p.getCount(name); got != 1 {
			t.Errorf("%v was read %v times, want once", name, got)
		}
	}
}
