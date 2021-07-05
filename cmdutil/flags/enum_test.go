// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package flags_test

import (
	"errors"
	"fmt"
	"testing"

	"cloudeng.io/cmdutil/flags"
)

func ExampleMap() {
	type dependEnum int
	const (
		Dependencies dependEnum = iota
		Dependents
	)
	mp := flags.Map{}.
		Register("dependencies", Dependencies).
		Register("dependents", Dependents).
		Default(Dependencies)

	if err := mp.Set("dependents"); err != nil {
		panic(err)
	}
	fmt.Println(mp.String())
	fmt.Println(mp.Get().(dependEnum))
	// Output:
	// dependents
	// 1
}

type dependEnum int

const (
	Dependencies dependEnum = iota
	Dependents
)

func TestMap(t *testing.T) {
	mp := flags.Map{}
	err := mp.Set("dependents")
	if got, want := mp.String(), ""; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	mp = mp.Default("bar")
	if got, want := mp.String(), "bar"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := mp.Get().(string), "bar"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if err == nil || !errors.Is(err, &flags.ErrMap{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
	mp = mp.Register("dependencies", Dependencies)
	err = mp.Set("xxx")
	t.Log(err)
	if err == nil || !errors.Is(err, &flags.ErrMap{}) {
		t.Errorf("missing or wrong error: %v", err)
	}
	if got, want := mp.String(), "bar"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	mp = mp.Register("dependents", Dependents)
	if err := mp.Set("dependents"); err != nil {
		t.Fatal(err)
	}
	if got, want := mp.Get().(dependEnum), Dependents; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := mp.String(), "dependents"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
