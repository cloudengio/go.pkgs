// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"context"
	"testing"

	"cloudeng.io/cmdutil"
)

type (
	configFile string
	envFile    string
)

func TestContextValue(t *testing.T) {
	ctx := context.Background()
	if _, ok := cmdutil.ContextValue[configFile](ctx); ok {
		t.Error("unexpected value")
	}
	ctx = cmdutil.ContextWithValue(ctx, configFile("a"))
	ctx = cmdutil.ContextWithValue(ctx, envFile("b"))
	ctx = cmdutil.ContextWithValue(ctx, []string{"c"})
	if v, ok := cmdutil.ContextValue[configFile](ctx); !ok || v != "a" {
		t.Errorf("got %v, %v", v, ok)
	}
	if v, ok := cmdutil.ContextValue[envFile](ctx); !ok || v != "b" {
		t.Errorf("got %v, %v", v, ok)
	}
	if v, ok := cmdutil.ContextValue[[]string](ctx); !ok || len(v) != 1 || v[0] != "c" {
		t.Errorf("got %v, %v", v, ok)
	}
	// A later value replaces an earlier one of the same type.
	ctx = cmdutil.ContextWithValue(ctx, configFile("z"))
	if v, _ := cmdutil.ContextValue[configFile](ctx); v != "z" {
		t.Errorf("got %v", v)
	}
	//nolint:staticcheck // a nil context is tolerated.
	if _, ok := cmdutil.ContextValue[configFile](nil); ok {
		t.Error("unexpected value")
	}
}
