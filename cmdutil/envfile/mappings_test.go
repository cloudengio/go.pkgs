// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/cmdutil/envfile"
)

func TestDefaultsMapping(t *testing.T) {
	defaults := map[string]string{"A": "a", "EMPTY": ""}
	mapping := envfile.DefaultsMapping(defaults)
	defaults["A"] = "changed" // the mapping has its own copy
	for name, want := range map[string]string{"A": "a", "EMPTY": "", "MISSING": ""} {
		if got := mapping(name); got != want {
			t.Errorf("%v: got %q, want %q", name, got, want)
		}
	}
	if got := envfile.DefaultsMapping(nil)("A"); got != "" {
		t.Errorf("nil defaults: got %q", got)
	}
}

func TestUnresolved(t *testing.T) {
	var unresolved envfile.Unresolved
	if err := unresolved.Err(); err != nil {
		t.Errorf("nothing recorded: got %v", err)
	}
	if names := unresolved.Names(); len(names) != 0 {
		t.Errorf("nothing recorded: got %v", names)
	}

	vars := envfile.NewVariables(map[string]string{"FROM_FILE": "file"})
	mapping := envfile.ChainMapping(
		vars.Mapping,
		envfile.DefaultsMapping(map[string]string{"DEFAULTED": "default", "FROM_FILE": "unused"}),
		unresolved.Mapping)
	got := os.Expand("$FROM_FILE ${DEFAULTED} $MISSING_B ${MISSING_A} $MISSING_B", mapping)
	if want := "file default   "; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := unresolved.Names(), []string{"MISSING_B", "MISSING_A"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Names: got %v, want %v", got, want)
	}

	err := unresolved.Err()
	if err == nil {
		t.Fatal("Err: got nil")
	}
	var ue *envfile.UnresolvedError
	if !errors.As(err, &ue) || ue.Name != "MISSING_B" {
		t.Errorf("errors.As: got %v", ue)
	}
	for _, name := range []string{"MISSING_A", "MISSING_B"} {
		if !strings.Contains(err.Error(), "variable "+name+" is not defined") {
			t.Errorf("Err: %q does not mention %v", err, name)
		}
	}
}

func TestUnresolvedConcurrent(t *testing.T) {
	var unresolved envfile.Unresolved
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for _, name := range []string{"A", "B", "C"} {
				unresolved.Mapping(name)
			}
		})
	}
	wg.Wait()
	if got := len(unresolved.Names()); got != 3 {
		t.Errorf("got %v names, want 3: %v", got, unresolved.Names())
	}
}
