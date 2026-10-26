// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/cmdyaml"
	"cloudeng.io/cmdutil/subcmd"
)

// ConfigHook is a PreHook, see its Run method, that loads the configuration
// of a command from the files named by a FilesFlag, registered as a global
// flag, and the environment variables files that go with them.
type ConfigHook[T any] struct {
	// ID identifies the hook in error reporting, it defaults to
	// "withConfigPrehook".
	ID string
	// Files is the flag that holds the names of the files.
	Files *FilesFlag
	// Spec determines how the files are located.
	Spec Spec
	// Initial is the value of the configuration before the files are parsed
	// into it, and the value that is used if there are none.
	Initial T
	// Required causes it to be an error for there to be no YAML file.
	Required bool
	// ParseOptions are passed to Load.
	ParseOptions []cmdyaml.Option
	// ValidateFields causes cmdyaml.ValidateFields to be called, before the
	// Validate method of the configuration, if it has one.
	ValidateFields bool
	// Store, if not nil, is called with the configuration, and the files it was
	// located in, to add anything else the commands need to the context. The
	// configuration, as a T, and the files, as a *Located, are always
	// added, see cmdutil.ContextWithValue.
	Store func(ctx context.Context, cfg T, loc *Located) context.Context
}

// Run locates and parses the files, as Load does, and validates the
// configuration if it was read from a file: the fields, if ValidateFields is set,
// and then the configuration itself if it has a Validate method. It adds the
// configuration and the files to the context and calls Store. Errors from
// parsing and validating say which files they are in and that they need to
// be edited. It implements PreHook.
func (h ConfigHook[T]) Run(ctx context.Context) (context.Context, string, subcmd.PostHook, error) {
	id := h.ID
	if id == "" {
		id = "withConfigPrehook"
	}
	postHook := func(context.Context) (string, error) { return id, nil }
	cfg := h.Initial
	loc, err := Load(ctx, &cfg, h.Files, h.Spec, h.ParseOptions...)
	if err != nil {
		if loc == nil {
			return ctx, id, postHook, err
		}
		return ctx, id, postHook, fmt.Errorf("failed to read the configuration file(s) %v: %w\nedit that file to correct it", strings.Join(loc.YAML, ", "), err)
	}
	if len(loc.YAML) == 0 && h.Required {
		return ctx, id, postHook, fmt.Errorf("no configuration file found, looked for %v in any of %v",
			strings.Join(h.Files.Files(FileTypeYAML, false), ", "), strings.Join(filepath.SplitList(loc.SearchPath), ", "))
	}
	if len(loc.YAML) > 0 {
		if err := h.validate(&cfg); err != nil {
			return ctx, id, postHook, fmt.Errorf("the configuration file(s) %v is invalid: %w\nedit that file to correct it", strings.Join(loc.YAML, ", "), err)
		}
	}
	ctx = cmdutil.ContextWithValue(ctx, cfg)
	ctx = cmdutil.ContextWithValue(ctx, loc)
	if h.Store != nil {
		ctx = h.Store(ctx, cfg, loc)
	}
	return ctx, id, postHook, nil
}

func (h ConfigHook[T]) validate(cfg *T) error {
	if h.ValidateFields {
		if err := cmdyaml.ValidateFields(cfg); err != nil {
			return err
		}
	}
	if v, ok := any(cfg).(interface{ Validate() error }); ok {
		return v.Validate()
	}
	return nil
}
