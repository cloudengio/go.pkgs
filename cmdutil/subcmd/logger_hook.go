// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"context"
	"flag"
	"fmt"

	"cloudeng.io/cmdutil"
	"cloudeng.io/logging/ctxlog"
)

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
