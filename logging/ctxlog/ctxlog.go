// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ctxlog provides a context key and functions for logging to a context.
package ctxlog

import (
	"context"

	"io"
	"log/slog"
)

type ctxKey struct{}

// NewJSONLogger returns a new context with a JSON logger.
func NewJSONLogger(ctx context.Context, w io.Writer, opts *slog.HandlerOptions) context.Context {
	return Context(ctx, slog.New(slog.NewJSONHandler(w, opts)))
}

// ContextWithLogger returns a new context with the given logger.
func Context(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey(struct{}{}), logger)
}

var discardLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

// LoggerFromContext returns the logger from the given context.
// If no logger is set, it returns a discard logger.
func Logger(ctx context.Context) *slog.Logger {
	l := ctx.Value(ctxKey(struct{}{}))
	if l == nil {
		return discardLogger
	}
	return l.(*slog.Logger)
}

// ContextWithLoggerAttributes returns a new context with the embedded logger
// updated with the given logger attributes.
func ContextWith(ctx context.Context, attributes ...any) context.Context {
	l := ctx.Value(ctxKey(struct{}{}))
	if l == nil {
		return ctx
	}
	return Context(ctx, l.(*slog.Logger).With(attributes...))
}
