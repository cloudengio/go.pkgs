// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxlog

import (
	"context"
	"log"
	"log/slog"
	"strings"
)

// CaptureLog redirects the standard library's default logger to write through
// the slog logger stored in ctx at the given level. Callers that use
// log.Print / log.Printf / log.Println will appear in the structured log
// stream alongside slog output.
//
// log's own date/time prefix flags are cleared because slog records the
// timestamp independently. The previous flags and output are not restored;
// call this once at program startup.
func CaptureLog(ctx context.Context, level slog.Level) {
	log.SetFlags(0)
	log.SetOutput(customLogWriter{ctx: context.WithoutCancel(ctx), level: level})
}

type customLogWriter struct {
	ctx   context.Context
	level slog.Level
}

func (c customLogWriter) Write(p []byte) (n int, err error) {
	// The standard logger outputs messages followed by a newline,
	// so trim it and log it the configured level.
	msg := strings.TrimSuffix(string(p), "\n")
	LogDepth(c.ctx, Logger(c.ctx), c.level, 5, msg)
	return len(p), nil
}

// NewLogLogger returns a new standard library logger that logs to the
// provided context's logger at the specified level.
func NewLogLogger(ctx context.Context, level slog.Level) *log.Logger {
	return log.New(customLogWriter{ctx: ctx, level: level}, "", 0)
}
