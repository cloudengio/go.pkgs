// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package path_test

import (
	"testing"

	"cloudeng.io/path"
)

func TestSharding(t *testing.T) {
	s1s := path.NewSharder(path.WithSHA1PrefixLength(3))
	p, s := s1s.Assign("abcded") // d550708ff9b78cac40527fdf4b237052d3a22f58
	if got, want := p, "d55"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := s, "0708ff9b78cac40527fdf4b237052d3a22f58"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	s1s = path.NewSharder()
	p, s = s1s.Assign("abcded") // d550708ff9b78cac40527fdf4b237052d3a22f58
	if got, want := p, "d"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := s, "550708ff9b78cac40527fdf4b237052d3a22f58"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
