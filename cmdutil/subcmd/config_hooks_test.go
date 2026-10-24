// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
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
	hook := subcmd.ConfigHook[hookConfig]{
		Files:    &globals.Config,
		Spec:     cmdconfig.Spec{AppDir: "hookstest"},
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
