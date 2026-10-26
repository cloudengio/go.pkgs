// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/cmdconfig"
)

type showConfig struct {
	Name string `yaml:"name" doc:"the name"`
	Port int    `yaml:"port" doc:"the port"`
}

func TestConfigCommandsShow(t *testing.T) {
	var out bytes.Buffer
	cc := cmdconfig.ConfigCommands[showConfig]{Output: &out}
	if err := cc.Show(context.Background(), nil, nil); err == nil {
		t.Error("expected an error when there is no configuration in the context")
	}

	ctx := cmdutil.ContextWithValue(context.Background(), showConfig{Name: "nm", Port: 8})
	if err := cc.Show(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "name: nm\nport: 8\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	out.Reset()
	ctx = cmdutil.ContextWithValue(ctx, &cmdconfig.Located{YAML: []string{"a.yaml", "b.yaml"}, Env: []string{"c.env"}})
	if err := cc.Show(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	want := "# Configuration file: a.yaml\n# Configuration file: b.yaml\n# Environment variables file: c.env\nname: nm\nport: 8\n"
	if got := out.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	out.Reset()
	ctx = cmdutil.ContextWithValue(ctx, &cmdconfig.Located{})
	if err := cc.Show(ctx, nil, nil); err != nil || !strings.HasPrefix(out.String(), "# Configuration file: none\n") {
		t.Errorf("got %q, %v", out.String(), err)
	}
}

func TestConfigCommandsDescribe(t *testing.T) {
	var out bytes.Buffer
	cc := cmdconfig.ConfigCommands[showConfig]{
		Summary: "show config options\n",
		Output:  &out,
		Describers: []func(io.Writer) error{
			func(w io.Writer) error { _, err := fmt.Fprintln(w, "extra documentation"); return err },
		},
	}
	if err := cc.Describe(context.Background(), nil, nil); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"show config options", "the name", "the port", "extra documentation"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q does not contain %q", got, want)
		}
	}
	if strings.Index(got, "the port") > strings.Index(got, "extra documentation") {
		t.Errorf("the extra documentation should follow: %q", got)
	}
}
