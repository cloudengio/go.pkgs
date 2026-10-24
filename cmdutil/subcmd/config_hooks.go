// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/cmdconfig"
	"cloudeng.io/cmdutil/cmdyaml"
	"cloudeng.io/logging/ctxlog"
)

// ConfigHook is a PreHook, see its Run method, that loads the configuration
// of a command from the files named by a FilesFlag, registered as a global
// flag, and the environment variables files that go with them.
type ConfigHook[T any] struct {
	// ID identifies the hook in error reporting, it defaults to
	// "withConfigPrehook".
	ID string
	// Files is the flag that holds the names of the files.
	Files *cmdconfig.FilesFlag
	// Spec determines how the files are located.
	Spec cmdconfig.Spec
	// Initial is the value of the configuration before the files are parsed
	// into it, and the value that is used if there are none.
	Initial T
	// Required causes it to be an error for there to be no YAML file.
	Required bool
	// ParseOptions are passed to cmdyaml.Load.
	ParseOptions []cmdyaml.Option
	// Store is called with the configuration, and the files it was located
	// in, to add them to the context for the commands.
	Store func(ctx context.Context, cfg T, loc *cmdconfig.Located) context.Context
}

// Run locates and parses the files, as cmdyaml.Load does, calls the Validate
// method of the configuration, if it has one and it was read from a file, and
// calls Store. Errors from
// parsing and validating say which files they are in and that they need to
// be edited. It implements PreHook.
func (h ConfigHook[T]) Run(ctx context.Context) (context.Context, string, PostHook, error) {
	id := h.ID
	if id == "" {
		id = "withConfigPrehook"
	}
	postHook := func(context.Context) (string, error) { return id, nil }
	cfg := h.Initial
	loc, err := cmdyaml.Load(ctx, &cfg, h.Files, h.Spec, h.ParseOptions...)
	if err != nil {
		if loc == nil {
			return ctx, id, postHook, err
		}
		return ctx, id, postHook, fmt.Errorf("failed to read the configuration file(s) %v: %w\nedit that file to correct it", strings.Join(loc.YAML, ", "), err)
	}
	if len(loc.YAML) == 0 && h.Required {
		return ctx, id, postHook, fmt.Errorf("no configuration file found, looked for %v in any of %v",
			strings.Join(h.Files.Files(cmdconfig.FileTypeYAML, false), ", "), strings.Join(filepath.SplitList(loc.SearchPath), ", "))
	}
	// The configuration is validated only if it was read from a file.
	if v, ok := any(&cfg).(interface{ Validate() error }); ok && len(loc.YAML) > 0 {
		if err := v.Validate(); err != nil {
			return ctx, id, postHook, fmt.Errorf("the configuration file(s) %v is invalid: %w\nedit that file to correct it", strings.Join(loc.YAML, ", "), err)
		}
	}
	if h.Store != nil {
		ctx = h.Store(ctx, cfg, loc)
	}
	return ctx, id, postHook, nil
}

// LoggerHook returns a PreHook that creates a logger using newLogger, adds it
// to the context for ctxlog.FromContext, and closes it after the command has
// run. It is typically registered after the hook that loads the configuration
// that newLogger takes the logging configuration from, together with the
// command line flags that override it, see cmdutil.LoggingConfig.WithFlagOverrides.
func LoggerHook(id string, newLogger func(ctx context.Context, fs *flag.FlagSet) (*cmdutil.Logger, error)) PreHook {
	return func(ctx context.Context) (context.Context, string, PostHook, error) {
		postHook := func(context.Context) (string, error) { return id, nil }
		global := GlobalFlagSetFromContext(ctx)
		if global == nil {
			return ctx, id, postHook, fmt.Errorf("global flagset not in context")
		}
		logger, err := newLogger(ctx, global.FlagSet())
		if err != nil {
			return ctx, id, postHook, fmt.Errorf("error setting up logger: %w", err)
		}
		postHook = func(context.Context) (string, error) {
			if err := logger.Close(); err != nil {
				return id, fmt.Errorf("error closing logger: %w", err)
			}
			return id, nil
		}
		return ctxlog.WithLogger(ctx, logger.Logger), id, postHook, nil
	}
}
