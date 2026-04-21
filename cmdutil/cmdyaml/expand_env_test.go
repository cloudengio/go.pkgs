// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
)

func envLookup(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestExpandEnv(t *testing.T) {
	env := envLookup(map[string]string{
		"HOST": "localhost",
		"PORT": "8080",
		"DIR":  "/tmp",
	})

	type simple struct {
		Addr   string `yaml:"addr"`
		NoTag  string
		NonStr int    `yaml:"non_str"`
		Dir    string `yaml:"dir"`
	}

	s := simple{Addr: "$HOST:$PORT", NoTag: "$HOST", NonStr: 42, Dir: "${DIR}"}
	cmdyaml.ExpandEnv(&s, env)
	if got, want := s.Addr, "localhost:8080"; got != want {
		t.Errorf("Addr: got %q, want %q", got, want)
	}
	if got, want := s.NoTag, "$HOST"; got != want {
		t.Errorf("NoTag: got %q, want %q (should be unchanged)", got, want)
	}
	if got, want := s.NonStr, 42; got != want {
		t.Errorf("NonStr: got %d, want %d", got, want)
	}
	if got, want := s.Dir, "/tmp"; got != want {
		t.Errorf("Dir: got %q, want %q", got, want)
	}
}

func TestExpandEnvEmbedded(t *testing.T) {
	env := envLookup(map[string]string{"A": "alpha", "B": "beta"})

	type inner struct {
		X string `yaml:"x"`
	}
	type outer struct {
		inner
		Y string `yaml:"y"`
	}

	o := outer{inner: inner{X: "$A"}, Y: "$B"}
	cmdyaml.ExpandEnv(&o, env)
	if got, want := o.X, "alpha"; got != want {
		t.Errorf("X: got %q, want %q", got, want)
	}
	if got, want := o.Y, "beta"; got != want {
		t.Errorf("Y: got %q, want %q", got, want)
	}
}

func TestExpandEnvNestedStruct(t *testing.T) {
	env := envLookup(map[string]string{"VAL": "expanded"})

	type leaf struct {
		V string `yaml:"v"`
	}
	type root struct {
		Sub leaf `yaml:"sub"`
	}

	r := root{Sub: leaf{V: "$VAL"}}
	cmdyaml.ExpandEnv(&r, env)
	if got, want := r.Sub.V, "expanded"; got != want {
		t.Errorf("Sub.V: got %q, want %q", got, want)
	}
}

func TestExpandEnvPointerEmbedded(t *testing.T) {
	env := envLookup(map[string]string{"Z": "zeta"})

	type inner struct {
		W string `yaml:"w"`
	}
	type outer struct {
		*inner
		Q string `yaml:"q"`
	}

	o := outer{inner: &inner{W: "$Z"}, Q: "$Z"}
	cmdyaml.ExpandEnv(&o, env)
	if got, want := o.W, "zeta"; got != want {
		t.Errorf("W: got %q, want %q", got, want)
	}
	if got, want := o.Q, "zeta"; got != want {
		t.Errorf("Q: got %q, want %q", got, want)
	}
}

func TestExpandEnvSlice(t *testing.T) {
	env := envLookup(map[string]string{"HOST": "localhost", "PORT": "8080"})

	type item struct {
		Addr string `yaml:"addr"`
	}
	type cfg struct {
		Items []item `yaml:"items"`
	}

	c := cfg{Items: []item{{"$HOST"}, {"$PORT"}}}
	cmdyaml.ExpandEnv(&c, env)
	if got, want := c.Items[0].Addr, "localhost"; got != want {
		t.Errorf("Items[0]: got %q, want %q", got, want)
	}
	if got, want := c.Items[1].Addr, "8080"; got != want {
		t.Errorf("Items[1]: got %q, want %q", got, want)
	}
}

func TestExpandEnvSliceOfStrings(t *testing.T) {
	env := envLookup(map[string]string{"A": "alpha", "B": "beta"})

	type cfg struct {
		Tags []string `yaml:"tags"`
	}

	c := cfg{Tags: []string{"$A", "$B", "literal"}}
	cmdyaml.ExpandEnv(&c, env)
	want := []string{"alpha", "beta", "literal"}
	for i, w := range want {
		if got := c.Tags[i]; got != w {
			t.Errorf("Tags[%d]: got %q, want %q", i, got, w)
		}
	}
}

func TestExpandEnvMap(t *testing.T) {
	env := envLookup(map[string]string{"VAL": "expanded"})

	type cfg struct {
		Vars map[string]string `yaml:"vars"`
	}

	c := cfg{Vars: map[string]string{"key": "$VAL", "other": "static"}}
	cmdyaml.ExpandEnv(&c, env)
	if got, want := c.Vars["key"], "expanded"; got != want {
		t.Errorf("Vars[key]: got %q, want %q", got, want)
	}
	if got, want := c.Vars["other"], "static"; got != want {
		t.Errorf("Vars[other]: got %q, want %q", got, want)
	}
}

func TestExpandEnvMapOfStructs(t *testing.T) {
	env := envLookup(map[string]string{"HOST": "localhost"})

	type server struct {
		Addr string `yaml:"addr"`
	}
	type cfg struct {
		Servers map[string]server `yaml:"servers"`
	}

	c := cfg{Servers: map[string]server{"web": {"$HOST"}, "api": {"static"}}}
	cmdyaml.ExpandEnv(&c, env)
	if got, want := c.Servers["web"].Addr, "localhost"; got != want {
		t.Errorf("Servers[web].Addr: got %q, want %q", got, want)
	}
	if got, want := c.Servers["api"].Addr, "static"; got != want {
		t.Errorf("Servers[api].Addr: got %q, want %q", got, want)
	}
}

func TestExpandEnvPointerToString(t *testing.T) {
	env := envLookup(map[string]string{"HOST": "localhost"})

	addr := "$HOST"
	type cfg struct {
		Addr     *string `yaml:"addr"`
		Optional *string `yaml:"optional"` // nil — must not panic
	}

	c := cfg{Addr: &addr, Optional: nil}
	cmdyaml.ExpandEnv(&c, env)
	if got, want := *c.Addr, "localhost"; got != want {
		t.Errorf("Addr: got %q, want %q", got, want)
	}
	if c.Optional != nil {
		t.Error("Optional: expected nil to remain nil")
	}
}

func TestExpandEnvSliceOfPointerToString(t *testing.T) {
	env := envLookup(map[string]string{"A": "alpha", "B": "beta"})

	a, b, lit := "$A", "$B", "literal"
	type cfg struct {
		Tags []*string `yaml:"tags"`
	}

	c := cfg{Tags: []*string{&a, &b, &lit}}
	cmdyaml.ExpandEnv(&c, env)
	want := []string{"alpha", "beta", "literal"}
	for i, w := range want {
		if got := *c.Tags[i]; got != w {
			t.Errorf("Tags[%d]: got %q, want %q", i, got, w)
		}
	}
}

func TestExpandEnvNilSafe(t *testing.T) {
	// nil pointer and non-struct inputs must not panic.
	cmdyaml.ExpandEnv(nil, func(string) string { return "" })

	var p *struct{ X string }
	cmdyaml.ExpandEnv(p, func(string) string { return "" })
}
