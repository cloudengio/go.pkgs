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
	testCases := []struct {
		name  string
		env   []string
		key   string
		value string
		want  []string
	}{
		{
			name:  "replace existing",
			env:   []string{"A=1", "B=2"},
			key:   "A",
			value: "3",
			want:  []string{"A=3", "B=2"},
		},
		{
			name:  "add new",
			env:   []string{"A=1", "B=2"},
			key:   "C",
			value: "4",
			want:  []string{"A=1", "B=2", "C=4"},
		},
		{
			name:  "add to empty",
			env:   []string{},
			key:   "A",
			value: "1",
			want:  []string{"A=1"},
		},
		{
			name:  "replace empty value",
			env:   []string{"A=", "B=2"},
			key:   "A",
			value: "3",
			want:  []string{"A=3", "B=2"},
		},
		{
			name:  "key is prefix of another key",
			env:   []string{"A=1", "AB=2"},
			key:   "A",
			value: "3",
			want:  []string{"A=3", "AB=2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The function can modify the slice in place, so make a copy.
			envCopy := make([]string, len(tc.env))
			copy(envCopy, tc.env)
			got := ReplaceEnvVar(envCopy, tc.key, tc.value)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%v: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
