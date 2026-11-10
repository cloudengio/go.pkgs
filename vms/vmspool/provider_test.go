// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool_test

import (
	"context"
	"testing"

	"cloudeng.io/vms/ociimage"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// TestPoolIdentity covers the name and size a pool is created with. Neither
// changes over its lifetime, which is why neither is reported by Stats, and
// both are answerable before Start has created anything.
func TestPoolIdentity(t *testing.T) {
	factory := vmstestutil.NewMockFactory(true)
	p := vmspool.New(factory, "macos-ci", vmspool.WithSize(3))
	if got, want := p.Name(), "macos-ci"; got != want {
		t.Errorf("Name: got %q, want %q", got, want)
	}
	if got, want := p.Size(), 3; got != want {
		t.Errorf("Size: got %v, want %v", got, want)
	}

	// An unset size falls back to the default rather than leaving a pool that
	// holds nothing.
	d := vmspool.New(factory, "defaults")
	if got, want := d.Size(), vmspool.DefaultPoolSize; got != want {
		t.Errorf("Size: got %v, want %v", got, want)
	}

	// The pool reports the provider it was created with, so an application
	// wrapping a pool reaches it through the pool rather than keeping a second
	// reference to it. It is answerable before the pool has created anything.
	if got, want := p.Provider(), vmspool.Provider(factory); got != want {
		t.Errorf("Provider: got %v, want %v", got, want)
	}
}

// TestVMProvenance covers what an acquired VM reports about where it came from:
// the pool that handed it over and the provider that created it. An
// application that runs several pools has only the VM to go on when it reports
// on a job, so this is how it tells them apart.
func TestVMProvenance(t *testing.T) {
	ctx := context.Background()
	factory := vmstestutil.NewMockFactory(true)
	image, err := ociimage.Parse("ghcr.io/cloudengio/ci-image:latest")
	if err != nil {
		t.Fatal(err)
	}
	factory.SetImage(image)
	p := newPool(t, 1, vmspool.StagingBehaviourSuspended, factory)

	vm, err := p.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if got, want := vm.PoolName(), testPoolName; got != want {
		t.Errorf("PoolName: got %q, want %q", got, want)
	}
	provider := vm.Provider()
	if provider == nil {
		t.Fatal("Provider: got nil, want the pool's provider")
	}
	if got, want := provider.Name(), vmstestutil.MockProviderName; got != want {
		t.Errorf("Provider.Name: got %q, want %q", got, want)
	}
	details := provider.Details()
	if details == nil {
		t.Fatal("Provider.Details: got nil, want the provider's details")
	}
	// The provider reports the image parsed, so a caller naming something
	// after it, or deciding whether it can be pulled, has the parts to hand.
	if got, want := details.Image(), image; got != want {
		t.Errorf("Provider.Details().Image: got %v, want %v", got, want)
	}
	if got, want := details.Image().Name, "ci-image"; got != want {
		t.Errorf("Provider.Details().Image().Name: got %q, want %q", got, want)
	}

	if err := vm.Delete(ctx); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

// TestVMNoPool covers a VM with no pool behind it, which is what a caller that
// kept a zero value, or read one from a map that had no entry, is holding.
// Reporting on it must not panic, since the point of these accessors is to
// describe a VM whose job has gone wrong.
func TestVMNoPool(t *testing.T) {
	var vm vmspool.VM
	if got := vm.PoolName(); got != "" {
		t.Errorf("PoolName: got %q, want %q", got, "")
	}
	if got := vm.Provider(); got != nil {
		t.Errorf("Provider: got %v, want nil", got)
	}
	if got := vm.ID(); got != "" {
		t.Errorf("ID: got %q, want %q", got, "")
	}
	if got := vm.CompletionQueue(); got != nil {
		t.Errorf("CompletionQueue: got %v, want nil", got)
	}
}

// TestTestVMPoolName covers the pool name reported by the VMs the stop tests
// build with NewTestVM, which have a closed pool behind them rather than a
// running one.
func TestTestVMPoolName(t *testing.T) {
	vm := vmspool.NewTestVM(vmstestutil.NewMock("vm-1"))
	if got, want := vm.PoolName(), vmspool.TestVMPoolName; got != want {
		t.Errorf("PoolName: got %q, want %q", got, want)
	}
}
