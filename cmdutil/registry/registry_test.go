// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package registry_test

import (
	"slices"
	"testing"

	"cloudeng.io/cmdutil/registry"
)

type Option func(o *int)

func TestGetOpts(t *testing.T) {
	opts := registry.ConvertAnyArgs[int](1, "a", 2, "b", 3.0)
	if got, want := len(opts), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := opts[0], 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := opts[1], 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	strs := registry.ConvertAnyArgs[string](1, "a", 2, "b", 3.0)
	if got, want := len(strs), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strs[0], "a"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strs[1], "b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	ofn := func(o *int) { *o = 2 }
	options := registry.ConvertAnyArgs[Option](1, "a", 2, Option(ofn), 3.0)
	if got, want := len(options), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := options[0]; got == nil {
		t.Errorf("got %v, want %v", got, nil)
	}
	var a, b int
	options[0](&a)
	options[0](&b)
	if got, want := a, b; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	none := registry.ConvertAnyArgs[bool](1, "a", 2, "b", 3.0)
	if got, want := len(none), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestKeys(t *testing.T) {
	r := &registry.T[any]{}
	if len(r.Keys()) != 0 {
		t.Errorf("expected empty slice for new registry, got %v", r.Keys())
	}
	r.Register("c", nil)
	r.Register("a", nil)
	r.Register("b", nil)

	keys := r.Keys()
	expected := []string{"a", "b", "c"}
	// The 'slices' package may need to be imported in the test file.
	if !slices.Equal(keys, expected) {
		t.Errorf("got %v, want %v", keys, expected)
	}
}
