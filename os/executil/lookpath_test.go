// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"reflect"
	"testing"
)

func TestReplaceEnvVar(t *testing.T) {
	t.Parallel()

	// Test replacing an existing variable.
	env := []string{"A=1", "B=2"}
	newEnv := ReplaceEnvVar(env, "A", "3")
	if got, want := newEnv, []string{"A=3", "B=2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Test adding a new variable.
	// Note that the original slice is modified when the key exists,
	// so we re-initialize it here.
	env = []string{"A=1", "B=2"}
	newEnv = ReplaceEnvVar(env, "C", "4")
	if got, want := newEnv, []string{"A=1", "B=2", "C=4"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Test adding a variable to an empty slice.
	env = []string{}
	newEnv = ReplaceEnvVar(env, "A", "1")
	if got, want := newEnv, []string{"A=1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
