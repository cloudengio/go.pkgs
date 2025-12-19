// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package registry_test

import (
	"testing"

	"cloudeng.io/cmdutil/registry"
)

func TestGetOpts(t *testing.T) {
	opts := registry.GetOpts[int](1, "a", 2, "b", 3.0)
	if got, want := len(opts), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := opts[0], 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := opts[1], 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	strs := registry.GetOpts[string](1, "a", 2, "b", 3.0)
	if got, want := len(strs), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strs[0], "a"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strs[1], "b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	floats := registry.GetOpts[float64](1, "a", 2, "b", 3.0)
	if got, want := len(floats), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := floats[0], 3.0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	none := registry.GetOpts[bool](1, "a", 2, "b", 3.0)
	if got, want := len(none), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
