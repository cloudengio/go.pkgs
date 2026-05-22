// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"context"
)

// PreHook represents a function that is called before the main command execution.
// It can modify the context and return a PostHook to be executed after the main
// command. An id is returned for inclusion in error reporting to allow for easy
// identification of the source of an error. PostHooks are executed in LIFO order
// (last registered, first called).
type PreHook func(context.Context) (ctx context.Context, id string, postHook PostHook, err error)

// PostHook represents a function that is called after the main command execution.
type PostHook func(context.Context) (id string, err error)
