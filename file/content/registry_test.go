// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content_test

import (
	"strings"
	"testing"

	"cloudeng.io/file/content"
)

func TestRegistry(t *testing.T) {
	reg := content.NewRegistry()
	var err error

	register := func(from, to string, handler interface{}) {
		if err = reg.Register(content.Type(from), content.Type(to), handler); err != nil {
			t.Fatal(err)
		}
	}

	lookup := func(from, to string, handler interface{}, par, val string) {
		p, v, got, err := reg.Lookup(content.Type(from), content.Type(to))
		if err != nil {
			t.Error(err)
		}
		if want := handler; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := p, par; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := v, val; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	h1 := &struct{}{}
	register("text/html", "text/plain", h1)
	lookup("text/html", "text/plain", h1, "", "")
	lookup("text/html;charset=utf-8", "text/plain", h1, "", "")
	lookup("text/html", "text/plain;charset=utf-8", h1, "charset", "utf-8")

	_, _, _, err = reg.Lookup("text/html", "text/plainx")
	if err == nil || !strings.Contains(err.Error(), "no handler for") {
		t.Fatal(err)
	}
	err = reg.Register("text/html", "text/plain", h1)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatal(err)
	}
}
