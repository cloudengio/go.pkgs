// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ctxlog provides a context key and functions for logging to a context.
package ctxlog

import (
	"context"
	"log"
	"runtime"
	"strings"
	"time"

	"io"
	"log/slog"
)

type ctxKey struct{}

// NewJSONLogger returns a new context with a JSON logger.
func NewJSONLogger(ctx context.Context, w io.Writer, opts *slog.HandlerOptions) context.Context {
	return WithLogger(ctx, slog.New(slog.NewJSONHandler(w, opts)))
}

// WithLogger returns a new context with the given logger.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey(struct{}{}), logger)
}

var discardLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

// Logger returns the logger from the given context.
// If no logger is set, it returns a discard logger.
func Logger(ctx context.Context) *slog.Logger {
	l := ctx.Value(ctxKey(struct{}{}))
	if l == nil {
		return discardLogger
	}
	return l.(*slog.Logger)
}

// WithAttributes returns a new context with the embedded logger
// updated with the given logger attributes.
func WithAttributes(ctx context.Context, attributes ...any) context.Context {
	l := ctx.Value(ctxKey(struct{}{}))
	if l == nil {
		return ctx
	}
	return WithLogger(ctx, l.(*slog.Logger).With(attributes...))
}

// LogDepth logs a message at the specified level with the caller
// information adjusted by the provided depth.
func LogDepth(ctx context.Context, logger *slog.Logger, level slog.Level, depth int, msg string, args ...any) {
	if !logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:]) // skip wrapper frames to get to the caller
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}

func Info(ctx context.Context, msg string, args ...any) {
	LogDepth(ctx, Logger(ctx), slog.LevelInfo, 3, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	LogDepth(ctx, Logger(ctx), slog.LevelError, 3, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	LogDepth(ctx, Logger(ctx), slog.LevelDebug, 3, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	LogDepth(ctx, Logger(ctx), slog.LevelWarn, 3, msg, args...)
}

func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	LogDepth(ctx, Logger(ctx), level, 3, msg, args...)
}

type customLogWriter struct {
	ctx   context.Context
	level slog.Level
}

func (c customLogWriter) Write(p []byte) (n int, err error) {
	// The standard logger outputs messages followed by a newline,
	// so trim it and log as an error.
	msg := strings.TrimSuffix(string(p), "\n")
	LogDepth(c.ctx, Logger(c.ctx), c.level, 5, msg)
	return len(p), nil
}

// NewLogLogger returns a new standard library logger that logs to the
// provided context's logger at the specified level.
func NewLogLogger(ctx context.Context, level slog.Level) *log.Logger {
	return log.New(customLogWriter{ctx: ctx, level: level}, "", 0)
}
