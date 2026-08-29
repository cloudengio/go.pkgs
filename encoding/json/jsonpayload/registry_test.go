// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"sync"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

type regA struct{ A int }

type regB struct{ B string }

func init() {
	jsonpayload.RegisterType[regA]()
	jsonpayload.RegisterType[regB]()
}

// TestRegistryNew verifies that a registered type is constructed as a pointer
// to a zero value of that type.
func TestRegistryNew(t *testing.T) {
	for _, tc := range []struct {
		name     string
		typeName string
		want     any
	}{
		{"regA", testPkg + ".regA", &regA{}},
		{"regB", testPkg + ".regB", &regB{}},
	} {
		v, ok := jsonpayload.New(tc.typeName)
		if !ok {
			t.Errorf("%v: %q is not registered", tc.name, tc.typeName)
			continue
		}
		if got, want := v, tc.want; !equalPointees(got, want) {
			t.Errorf("%v: got %T %+v, want %T %+v", tc.name, got, got, want, want)
		}
	}
}

func equalPointees(got, want any) bool {
	switch w := want.(type) {
	case *regA:
		g, ok := got.(*regA)
		return ok && *g == *w
	case *regB:
		g, ok := got.(*regB)
		return ok && *g == *w
	}
	return false
}

// TestRegistryUnknownType verifies the lookup miss path.
func TestRegistryUnknownType(t *testing.T) {
	for _, name := range []string{
		testPkg + ".regUnregistered",
		"no.such.Type",
		"",
	} {
		if v, ok := jsonpayload.New(name); ok || v != nil {
			t.Errorf("%q: got (%v, %v), want (nil, false)", name, v, ok)
		}
	}
}

// TestRegistryNewIsFresh verifies that each call constructs a new value:
// decoded messages must not share state with each other.
func TestRegistryNewIsFresh(t *testing.T) {
	first, ok := jsonpayload.New(testPkg + ".regA")
	if !ok {
		t.Fatal("regA is not registered")
	}
	second, ok := jsonpayload.New(testPkg + ".regA")
	if !ok {
		t.Fatal("regA is not registered")
	}
	if first == second {
		t.Fatal("New returned the same instance twice")
	}
	first.(*regA).A = 42
	if got := second.(*regA).A; got != 0 {
		t.Errorf("instances share state: got %v, want 0", got)
	}
}

// TestRegistryReRegister verifies that registering a type more than once is
// harmless, which matters because registration usually happens in init.
func TestRegistryReRegister(t *testing.T) {
	jsonpayload.RegisterType[regA]()
	jsonpayload.RegisterType[regA]()
	v, ok := jsonpayload.New(testPkg + ".regA")
	if !ok {
		t.Fatal("regA is not registered")
	}
	if _, ok := v.(*regA); !ok {
		t.Errorf("got %T, want *regA", v)
	}
}

// TestRegistryConcurrent exercises the registry's concurrent Store and Load
// paths; run under -race it verifies that registration and lookup are safe to
// interleave.
func TestRegistryConcurrent(t *testing.T) {
	const goroutines = 8
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for range 100 {
				jsonpayload.RegisterType[regA]()
				v, ok := jsonpayload.New(testPkg + ".regA")
				if !ok {
					t.Error("regA is not registered")
					return
				}
				if _, ok := v.(*regA); !ok {
					t.Errorf("got %T, want *regA", v)
					return
				}
			}
		})
	}
	wg.Wait()
}
