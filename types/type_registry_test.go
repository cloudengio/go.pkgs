// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package types_test

import (
	"fmt"
	"sync"
	"testing"

	"cloudeng.io/types"
)

// TestRegistryRegisterAndNew verifies that a registered type is keyed by its
// type name and constructed as a pointer to a zero value.
func TestRegistryRegisterAndNew(t *testing.T) {
	var r types.Registry
	r.RegisterType[myStruct]()

	v, ok := r.New(pkg + ".myStruct")
	if !ok {
		t.Fatalf("%v.myStruct is not registered", pkg)
	}
	p, ok := v.(*myStruct)
	if !ok {
		t.Fatalf("got %T, want *myStruct", v)
	}
	if p == nil {
		t.Fatal("got a nil pointer")
	}
	if got, want := *p, (myStruct{}); got != want {
		t.Errorf("got %+v, want the zero value %+v", got, want)
	}
}

// TestRegistryMultipleTypes verifies that one registry holds any number of
// types, each keyed by its own name.
func TestRegistryMultipleTypes(t *testing.T) {
	var r types.Registry
	r.RegisterType[myStruct]()
	r.RegisterType[myInt]()
	r.RegisterType[mySlice]()
	r.RegisterType[myGeneric[int]]()
	r.RegisterType[[]int]()

	for _, tc := range []struct {
		name string
		want any
	}{
		{pkg + ".myStruct", &myStruct{}},
		{pkg + ".myInt", new(myInt)},
		{pkg + ".mySlice", &mySlice{}},
		{pkg + ".myGeneric[int]", &myGeneric[int]{}},
		{"[]int", &[]int{}},
	} {
		v, ok := r.New(tc.name)
		if !ok {
			t.Errorf("%v is not registered", tc.name)
			continue
		}
		if got, want := typeOfValue(v), typeOfValue(tc.want); got != want {
			t.Errorf("%v: got %v, want %v", tc.name, got, want)
		}
	}
}

// typeOfValue reports the dynamic type of v using fmt rather than this
// package's own TypeName, so that the registry is checked independently of the
// naming it is keyed by.
func typeOfValue(v any) string {
	return fmt.Sprintf("%T", v)
}

// TestRegistryUnknownType covers the lookup miss path.
func TestRegistryUnknownType(t *testing.T) {
	var r types.Registry
	r.RegisterType[myStruct]()

	for _, name := range []string{
		pkg + ".myInt", // a real type, but not registered here
		"no.such.Type",
		"myStruct", // unqualified
		"",
	} {
		if v, ok := r.New(name); ok || v != nil {
			t.Errorf("%q: got (%v, %v), want (nil, false)", name, v, ok)
		}
	}
}

// TestRegistryZeroValue verifies that the zero value is usable without any
// construction, and that it starts out empty.
func TestRegistryZeroValue(t *testing.T) {
	var r types.Registry
	if v, ok := r.New(pkg + ".myStruct"); ok || v != nil {
		t.Errorf("got (%v, %v), want (nil, false) from an empty registry", v, ok)
	}
	r.RegisterType[myStruct]()
	if _, ok := r.New(pkg + ".myStruct"); !ok {
		t.Error("myStruct is not registered after RegisterType on a zero value")
	}
}

// TestRegistryNewIsFresh verifies that each call constructs a new value, so
// that values handed out by the registry never share state.
func TestRegistryNewIsFresh(t *testing.T) {
	var r types.Registry
	r.RegisterType[myStruct]()

	first, ok := r.New(pkg + ".myStruct")
	if !ok {
		t.Fatal("myStruct is not registered")
	}
	second, ok := r.New(pkg + ".myStruct")
	if !ok {
		t.Fatal("myStruct is not registered")
	}
	if first == second {
		t.Fatal("New returned the same instance twice")
	}
	first.(*myStruct).A = 42
	if got := second.(*myStruct).A; got != 0 {
		t.Errorf("instances share state: got %v, want 0", got)
	}
}

// TestRegistryReRegister verifies that registering more than once is harmless,
// since registration is typically done from an init function.
func TestRegistryReRegister(t *testing.T) {
	var r types.Registry
	r.RegisterType[myStruct]()
	r.RegisterType[myStruct]()

	v, ok := r.New(pkg + ".myStruct")
	if !ok {
		t.Fatal("myStruct is not registered")
	}
	if _, ok := v.(*myStruct); !ok {
		t.Errorf("got %T, want *myStruct", v)
	}
}

// TestRegistryInstancesAreIndependent verifies that each registry owns its own
// contents, so that registering with one does not affect another.
func TestRegistryInstancesAreIndependent(t *testing.T) {
	var registered, empty types.Registry
	registered.RegisterType[myStruct]()

	if _, ok := registered.New(pkg + ".myStruct"); !ok {
		t.Error("myStruct is not registered in the registry it was registered with")
	}
	if v, ok := empty.New(pkg + ".myStruct"); ok || v != nil {
		t.Errorf("got (%v, %v), want (nil, false) from a separate registry", v, ok)
	}
}

// TestRegistryConcurrent exercises the concurrent Store and Load paths; run
// under -race it verifies that registration and lookup of several types can be
// interleaved.
func TestRegistryConcurrent(t *testing.T) {
	var r types.Registry
	const goroutines = 8
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range 100 {
				r.RegisterType[myStruct]()
				r.RegisterType[myInt]()
				v, ok := r.New(pkg + ".myStruct")
				if !ok {
					t.Error("myStruct is not registered")
					return
				}
				if _, ok := v.(*myStruct); !ok {
					t.Errorf("got %T, want *myStruct", v)
					return
				}
				if _, ok := r.New(pkg + ".myInt"); !ok {
					t.Error("myInt is not registered")
					return
				}
			}
		})
	}
	wg.Wait()
}
