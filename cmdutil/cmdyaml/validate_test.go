// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"errors"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
)

type valueV struct{ Bad bool }

func (v valueV) Validate() error {
	if v.Bad {
		return errors.New("value is bad")
	}
	return nil
}

type pointerV struct{ Bad bool }

func (v *pointerV) Validate() error {
	if v.Bad {
		return errors.New("pointer is bad")
	}
	return nil
}

type ownsChildren struct {
	Child valueV `yaml:"child"`
}

func (o ownsChildren) Validate() error { return nil }

type Inlined struct {
	Inner valueV `yaml:"inner"`
}

type sample struct {
	Inlined `yaml:",inline"`
	Name    string            `yaml:"name"`
	Value   valueV            `yaml:"value"`
	Ptr     *valueV           `yaml:"ptr"`
	PtrRecv pointerV          `yaml:"ptr_recv"`
	List    []valueV          `yaml:"list"`
	Map     map[string]valueV `yaml:"map"`
	Iface   any               `yaml:"iface"`
	Owned   ownsChildren      `yaml:"owned"`
	NoTag   valueV
	Nil     *valueV `yaml:"nil"`
	private valueV
	Nested  struct{ In valueV } `yaml:"nested"`
}

func (s sample) Validate() error { return errors.New("root Validate must not be called") }

func TestValidateFields(t *testing.T) {
	if err := cmdyaml.ValidateFields(&sample{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tc := range []struct {
		name string
		mod  func(*sample)
		want string
	}{
		{"value", func(s *sample) { s.Value.Bad = true }, "value: value is bad"},
		{"inline", func(s *sample) { s.Inner.Bad = true }, "inner: value is bad"},
		{"pointer", func(s *sample) { s.Ptr = &valueV{Bad: true} }, "ptr: value is bad"},
		{"pointer receiver", func(s *sample) { s.PtrRecv.Bad = true }, "ptr_recv: pointer is bad"},
		{"list", func(s *sample) { s.List = []valueV{{}, {Bad: true}} }, "list[1]: value is bad"},
		{"map", func(s *sample) { s.Map = map[string]valueV{"k": {Bad: true}} }, "map[k]: value is bad"},
		{"interface", func(s *sample) { s.Iface = valueV{Bad: true} }, "iface: value is bad"},
		{"untagged", func(s *sample) { s.NoTag.Bad = true }, "notag: value is bad"},
		{"nested", func(s *sample) { s.Nested.In.Bad = true }, "nested.in: value is bad"},
	} {
		var s sample
		tc.mod(&s)
		if err := cmdyaml.ValidateFields(&s); err == nil || err.Error() != tc.want {
			t.Errorf("%v: got %v, want %q", tc.name, err, tc.want)
		}
	}
	// A value with its own Validate is responsible for its fields; the
	// unexported and nil ones are skipped.
	s := sample{Owned: ownsChildren{Child: valueV{Bad: true}}, private: valueV{Bad: true}}
	if err := cmdyaml.ValidateFields(&s); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := cmdyaml.ValidateFields(3); err == nil || !strings.Contains(err.Error(), "not a struct") {
		t.Errorf("unexpected error: %v", err)
	}
}
