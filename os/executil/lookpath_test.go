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

func TestGetenv(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		env  []string
		key  string
		want string
		ok   bool
	}{
		{
			name: "exists",
			env:  []string{"A=1", "B=2"},
			key:  "A",
			want: "1",
			ok:   true,
		},
		{
			name: "does not exist",
			env:  []string{"A=1", "B=2"},
			key:  "C",
			want: "",
			ok:   false,
		},
		{
			name: "empty value",
			env:  []string{"A=", "B=2"},
			key:  "A",
			want: "",
			ok:   true,
		},
		{
			name: "empty env",
			env:  []string{},
			key:  "A",
			want: "",
			ok:   false,
		},
		{
			name: "key is prefix of another key",
			env:  []string{"A=1", "AB=2"},
			key:  "A",
			want: "1",
			ok:   true,
		},
		{
			name: "key is prefix of another key but no exact match",
			env:  []string{"AB=2"},
			key:  "A",
			want: "",
			ok:   false,
		},
		{
			name: "malformed entry without equals",
			env:  []string{"A", "B=2"},
			key:  "A",
			want: "",
			ok:   false,
		},
		{
			name: "empty key",
			env:  []string{"=1", "A=2"},
			key:  "",
			want: "",
			ok:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			val, ok := Getenv(tc.env, tc.key)
			if got, want := ok, tc.ok; got != want {
				t.Errorf("%v: got %v, want %v", tc.name, got, want)
			}
			if got, want := val, tc.want; got != want {
				t.Errorf("%v: got %v, want %v", tc.name, got, want)
			}
		})
	}
}
