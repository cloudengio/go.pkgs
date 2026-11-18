// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig_test

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/cmdconfig"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/logging/ctxlog"
)

type hookConfig struct {
	Name    string `yaml:"name"`
	Logging cmdutil.LoggingConfig
}

func (c *hookConfig) Validate() error {
	if c.Name == "invalid" {
		return os.ErrInvalid
	}
	return nil
}

type hookGlobals struct {
	Config cmdconfig.FilesFlag `subcmd:"config,,configuration files"`
	cmdutil.LoggingFlags
}

type storedConfig struct{}

func runHooks(t *testing.T, dir string, required bool, args ...string) (hookConfig, *cmdconfig.Located, context.Context, error) {
	t.Helper()
	t.Chdir(dir)
	var globals hookGlobals
	globals.Config.SetDefaults("hooks.yaml")
	fs := subcmd.NewFlagSet().MustRegisterFlagStruct(&globals, nil, nil)
	var got hookConfig
	var loc *cmdconfig.Located
	var runCtx context.Context
	hook := cmdconfig.ConfigHook[hookConfig]{
		Files:    &globals.Config,
		Spec:     cmdconfig.Spec{Path: cmdconfig.SearchPath("hookstest")},
		Initial:  hookConfig{Name: "initial"},
		Required: required,
		Store: func(ctx context.Context, cfg hookConfig, l *cmdconfig.Located) context.Context {
			got, loc = cfg, l
			return cmdutil.ContextWithValue(ctx, storedConfig{})
		},
	}
	logger := subcmd.LoggerHook("logger", func(ctx context.Context, fs *flag.FlagSet) (*cmdutil.Logger, error) {
		if _, ok := cmdutil.ContextValue[storedConfig](ctx); !ok {
			t.Error("the logger hook should run after the configuration hook")
		}
		return got.Logging.WithFlagOverrides(fs, globals.LoggingFlags).NewLogger()
	})
	runner := func(ctx context.Context, _ any, _ []string) error {
		runCtx = ctx
		return nil
	}
	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(hook.Run, logger),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	cs.WithGlobalFlags(fs)
	err := cs.DispatchWithArgs(context.Background(), "test", append(args, "cmd")...)
	return got, loc, runCtx, err
}

func TestConfigHook(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hooks.yaml"), []byte("name: from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.yaml"), []byte("name: other\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// The default.
	cfg, loc, ctx, err := runHooks(t, dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "from-file" || len(loc.YAML) != 1 {
		t.Errorf("got %+v, %+v", cfg, loc)
	}
	if _, ok := cmdutil.ContextValue[storedConfig](ctx); !ok {
		t.Error("the context returned by Store was not used")
	}
	if ctxlog.Logger(ctx) == nil {
		t.Error("expected a logger in the context")
	}
	// From the command line.
	cfg, _, _, err = runHooks(t, dir, true, "--config=other.yaml")
	if err != nil || cfg.Name != "other" {
		t.Errorf("got %+v, %v", cfg, err)
	}
}

func TestConfigHookErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "invalid.yaml"), []byte("name: invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unknown.yaml"), []byte("nope: 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args     []string
		required bool
		want     string
	}{
		{[]string{"--config=missing.yaml"}, true, "missing.yaml"},
		{[]string{"--config=invalid.yaml"}, true, "is invalid"},
		{[]string{"--config=unknown.yaml"}, true, "edit that file to correct it"},
		{nil, true, "no configuration file found"},
	} {
		if _, _, _, err := runHooks(t, dir, tc.required, tc.args...); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want %q", tc.args, err, tc.want)
		}
	}
	// Optional: the initial value is used.
	cfg, loc, _, err := runHooks(t, dir, false)
	if err != nil || cfg.Name != "initial" || len(loc.YAML) != 0 {
		t.Errorf("got %+v, %+v, %v", cfg, loc, err)
	}
}

type validatedField struct{ Bad bool }

func (v validatedField) Validate() error {
	if v.Bad {
		return os.ErrInvalid
	}
	return nil
}

type fieldsConfig struct {
	Field validatedField `yaml:"field"`
}

func TestConfigHookValidateFields(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "fields.yaml"), []byte("field:\n  bad: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(validateFields bool) (context.Context, error) {
		var ff cmdconfig.FilesFlag
		ff.SetDefaults("fields.yaml")
		ctx, _, _, err := cmdconfig.ConfigHook[fieldsConfig]{
			Files:          &ff,
			Spec:           cmdconfig.Spec{Path: cmdconfig.SearchPath("hookstest")},
			ValidateFields: validateFields,
		}.Run(context.Background())
		return ctx, err
	}
	// The walk is opt-in.
	ctx, err := run(false)
	if err != nil {
		t.Fatal(err)
	}
	// The configuration and the files are always added to the context.
	if cfg, ok := cmdutil.ContextValue[fieldsConfig](ctx); !ok || !cfg.Field.Bad {
		t.Errorf("got %+v, %v", cfg, ok)
	}
	if loc, ok := cmdutil.ContextValue[*cmdconfig.Located](ctx); !ok || len(loc.YAML) != 1 {
		t.Errorf("got %+v, %v", loc, ok)
	}
	if _, err = run(true); err == nil || !strings.Contains(err.Error(), "field: invalid argument") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConfigExtension(t *testing.T) {
	const spec = `name: tool
summary: a tool
commands:
  - name: run
    summary: run the tool
  {{range subcmdExtension "config"}}{{.}}
  {{end}}
`
	var out bytes.Buffer
	cc := cmdconfig.ConfigCommands[hookConfig]{Output: &out, Summary: "tool options\n"}
	cmdSet := subcmd.MustFromYAMLTemplate(spec, cmdconfig.ConfigExtension(cc))
	cmdSet.MustAddExtensions()
	cmdSet.Set("run").MustRunner(func(context.Context, any, []string) error { return nil }, &struct{}{})

	ctx := cmdutil.ContextWithValue(context.Background(), hookConfig{Name: "shown"})
	if err := cmdSet.DispatchWithArgs(ctx, "tool", "config", "show"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "name: shown") {
		t.Errorf("got %q", out.String())
	}
	out.Reset()
	if err := cmdSet.DispatchWithArgs(ctx, "tool", "config", "describe"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "tool options") {
		t.Errorf("got %q", out.String())
	}
	if err := cmdSet.DispatchWithArgs(ctx, "tool", "config", "edit"); err == nil {
		t.Errorf("expected config edit to be omitted when Editor is nil")
	}
}

func TestConfigExtensionEdit(t *testing.T) {
	const spec = `name: tool
summary: a tool
commands:
  {{range subcmdExtension "config"}}{{.}}
  {{end}}
`
	type editFlags struct {
		Force bool `subcmd:"force,false,force edit"`
	}

	t.Run("default flags", func(t *testing.T) {
		var gotArgs []string
		cc := cmdconfig.ConfigCommands[hookConfig]{
			Editor: func(_ context.Context, _ any, args []string) error {
				gotArgs = args
				return nil
			},
		}
		cmdSet := subcmd.MustFromYAMLTemplate(spec, cmdconfig.ConfigExtension(cc))
		cmdSet.MustAddExtensions()

		if err := cmdSet.DispatchWithArgs(context.Background(), "tool", "config", "edit", "a.yaml", "b.yaml"); err != nil {
			t.Fatal(err)
		}
		if len(gotArgs) != 2 || gotArgs[0] != "a.yaml" || gotArgs[1] != "b.yaml" {
			t.Errorf("got args %v, want [a.yaml b.yaml]", gotArgs)
		}
	})

	t.Run("custom flags", func(t *testing.T) {
		var gotForce bool
		var gotArgs []string
		cc := cmdconfig.ConfigCommands[hookConfig]{
			EditFlags: &editFlags{},
			Editor: func(_ context.Context, fv any, args []string) error {
				gotForce = fv.(*editFlags).Force
				gotArgs = args
				return nil
			},
		}
		cmdSet := subcmd.MustFromYAMLTemplate(spec, cmdconfig.ConfigExtension(cc))
		cmdSet.MustAddExtensions()

		if err := cmdSet.DispatchWithArgs(context.Background(), "tool", "config", "edit", "--force", "cfg.yaml"); err != nil {
			t.Fatal(err)
		}
		if !gotForce || len(gotArgs) != 1 || gotArgs[0] != "cfg.yaml" {
			t.Errorf("got force=%v args=%v, want force=true args=[cfg.yaml]", gotForce, gotArgs)
		}
	})
}
