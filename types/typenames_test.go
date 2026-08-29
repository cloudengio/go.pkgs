// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package types_test

import (
	"bytes"
	"errors"
	"sync"
	"testing"
	"time"

	"cloudeng.io/types"
)

// pkg is the path of this external test package; the named types declared
// below are all reported as being within it.
const pkg = "cloudeng.io/types_test"

type myStruct struct{ A int }

type myInt int

type mySlice []int

type myIface interface{ Method() }

type myGeneric[T any] struct{ V T }

func TestTypeName(t *testing.T) {
	for _, tc := range []struct {
		name, got, want string
	}{
		// Named types declared in this package.
		{"struct", types.TypeName[myStruct](), pkg + ".myStruct"},
		{"named int", types.TypeName[myInt](), pkg + ".myInt"},
		{"named slice", types.TypeName[mySlice](), pkg + ".mySlice"},
		{"named interface", types.TypeName[myIface](), pkg + ".myIface"},

		// Predeclared types are not qualified since they have no package.
		{"int", types.TypeName[int](), "int"},
		{"string", types.TypeName[string](), "string"},
		{"float64", types.TypeName[float64](), "float64"},
		{"error", types.TypeName[error](), "error"},

		// Types declared in other packages.
		{"time.Time", types.TypeName[time.Time](), "time.Time"},
		{"bytes.Buffer", types.TypeName[bytes.Buffer](), "bytes.Buffer"},
	} {
		if tc.got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestTypeNamePointerIndirection verifies that pointer indirection is removed,
// to any depth, so that a type and pointers to it share a single name.
func TestTypeNamePointerIndirection(t *testing.T) {
	for _, tc := range []struct {
		name, got, want string
	}{
		{"pointer to struct", types.TypeName[*myStruct](), pkg + ".myStruct"},
		{"pointer to pointer", types.TypeName[**myStruct](), pkg + ".myStruct"},
		{"pointer to named int", types.TypeName[*myInt](), pkg + ".myInt"},
		{"pointer to bytes.Buffer", types.TypeName[*bytes.Buffer](), "bytes.Buffer"},
		{"pointer to int", types.TypeName[*int](), "int"},
	} {
		if tc.got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestTypeNameUnnamedTypes documents that a composite type with no name of its
// own is reported as the empty string: there is nothing to qualify. Note that
// this means all such types share a single name.
func TestTypeNameUnnamedTypes(t *testing.T) {
	for _, tc := range []struct {
		name, got string
	}{
		{"slice", types.TypeName[[]int]()},
		{"map", types.TypeName[map[string]int]()},
		{"array", types.TypeName[[3]int]()},
		{"channel", types.TypeName[chan int]()},
		{"func", types.TypeName[func() error]()},
		{"anonymous struct", types.TypeName[struct{ A int }]()},
		{"any", types.TypeName[any]()},
		// Only pointer indirection is removed, so the element type of a
		// composite is not consulted.
		{"slice of pointers", types.TypeName[[]*myStruct]()},
		{"pointer to slice", types.TypeName[*[]int]()},
	} {
		if tc.got != "" {
			t.Errorf("%v: got %q, want the empty string", tc.name, tc.got)
		}
	}
}

// TestTypeNameGeneric verifies that an instantiated generic type reports its
// type arguments as part of its name.
func TestTypeNameGeneric(t *testing.T) {
	for _, tc := range []struct {
		name, got, want string
	}{
		{"builtin argument", types.TypeName[myGeneric[int]](), pkg + ".myGeneric[int]"},
		{"named argument", types.TypeName[myGeneric[myStruct]](), pkg + ".myGeneric[" + pkg + ".myStruct]"},
		{"pointer to generic", types.TypeName[*myGeneric[int]](), pkg + ".myGeneric[int]"},
	} {
		if tc.got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestTypeNameForValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    any
		want string
	}{
		{"struct", myStruct{}, pkg + ".myStruct"},
		{"pointer to struct", &myStruct{}, pkg + ".myStruct"},
		{"named int", myInt(3), pkg + ".myInt"},
		{"named slice", mySlice{1, 2}, pkg + ".mySlice"},

		{"int", 42, "int"},
		{"string", "hello", "string"},

		{"time.Time", time.Time{}, "time.Time"},
		{"pointer to bytes.Buffer", &bytes.Buffer{}, "bytes.Buffer"},

		// The dynamic type of the value is reported, not the static type of
		// whatever interface it was passed as.
		{"error value", errors.New("an error"), "errors.errorString"},

		// As for TypeName, unnamed composite types have no name to report.
		{"slice", []int{1, 2}, ""},
		{"map", map[string]int{"a": 1}, ""},
		{"anonymous struct", struct{ A int }{A: 1}, ""},
	} {
		if got, want := types.TypeNameForValue(tc.v), tc.want; got != want {
			t.Errorf("%v: got %q, want %q", tc.name, got, want)
		}
	}
}

// TestTypeNameForValueNil covers the values that carry no type, or a type but
// no value.
func TestTypeNameForValueNil(t *testing.T) {
	// An untyped nil carries no type at all.
	if got := types.TypeNameForValue(nil); got != "" {
		t.Errorf("untyped nil: got %q, want the empty string", got)
	}

	// A typed nil still reports its type: the type is carried by the
	// interface rather than by the value.
	var p *myStruct
	if got, want := types.TypeNameForValue(p), pkg+".myStruct"; got != want {
		t.Errorf("nil pointer: got %q, want %q", got, want)
	}
	var pp **myStruct
	if got, want := types.TypeNameForValue(pp), pkg+".myStruct"; got != want {
		t.Errorf("nil pointer to pointer: got %q, want %q", got, want)
	}
	var s mySlice
	if got, want := types.TypeNameForValue(s), pkg+".mySlice"; got != want {
		t.Errorf("nil named slice: got %q, want %q", got, want)
	}
	var err error
	if got := types.TypeNameForValue(err); got != "" {
		t.Errorf("nil error: got %q, want the empty string", got)
	}
}

// TestTypeNameConsistency verifies that the two forms agree with each other,
// including across the pointer indirection that both remove.
func TestTypeNameConsistency(t *testing.T) {
	for _, tc := range []struct {
		name, fromType, fromValue string
	}{
		{"struct", types.TypeName[myStruct](), types.TypeNameForValue(myStruct{})},
		{"pointer", types.TypeName[*myStruct](), types.TypeNameForValue(&myStruct{})},
		{"value and pointer", types.TypeName[myStruct](), types.TypeNameForValue(&myStruct{})},
		{"named int", types.TypeName[myInt](), types.TypeNameForValue(myInt(0))},
		{"int", types.TypeName[int](), types.TypeNameForValue(0)},
		{"time.Time", types.TypeName[time.Time](), types.TypeNameForValue(time.Time{})},
		{"slice", types.TypeName[[]int](), types.TypeNameForValue([]int{})},
	} {
		if tc.fromType != tc.fromValue {
			t.Errorf("%v: TypeName gave %q, TypeNameForValue gave %q", tc.name, tc.fromType, tc.fromValue)
		}
	}
}

// TestTypeNameCaching verifies that repeated and concurrent lookups all agree;
// run under -race it exercises the concurrent Load and Store paths of the
// shared cache.
func TestTypeNameCaching(t *testing.T) {
	want := [3]string{pkg + ".myStruct", "int", "time.Time"}

	// A first, single-threaded pass populates the cache, so the goroutines
	// below exercise both the populated and unpopulated paths.
	if got := [3]string{
		types.TypeName[myStruct](),
		types.TypeName[int](),
		types.TypeNameForValue(time.Time{}),
	}; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	const goroutines = 8
	got := make([][3]string, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			for range 50 {
				got[i] = [3]string{
					types.TypeName[myStruct](),
					types.TypeName[int](),
					types.TypeNameForValue(time.Time{}),
				}
			}
		})
	}
	wg.Wait()
	for i, g := range got {
		if g != want {
			t.Errorf("goroutine %v: got %v, want %v", i, g, want)
		}
	}
}
