// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"runtime"
	"testing"

	"cloudeng.io/logging/ctxlog"
)

func TestContextLogger(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	// Test JSON logger creation and retrieval
	jsonCtx := ctxlog.NewJSONLogger(ctx, buf, nil)
	logger := ctxlog.Logger(jsonCtx)
	if logger == nil {
		t.Error("expected non-nil logger")
	}

	// Test logging works
	logger.Info("test message", "key", "value")
	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Error("expected log message to be written")
	}

	// Test context with attributes
	attrCtx := ctxlog.WithAttributes(jsonCtx, "attr1", "val1")
	attrLogger := ctxlog.Logger(attrCtx)
	attrLogger.Info("test")
	if !bytes.Contains(buf.Bytes(), []byte("attr1")) {
		t.Error("expected attribute in log output")
	}

	// Test nil logger returns discard logger
	nilLogger := ctxlog.Logger(ctx)
	if nilLogger == nil {
		t.Error("expected non-nil discard logger")
	}
}

func ExampleLogger() {
	// Create a context with a JSON logger
	ctx := context.Background()
	buf := &bytes.Buffer{}
	ctx = ctxlog.NewJSONLogger(ctx, buf, nil)

	// Get logger from context and use it
	logger := ctxlog.Logger(ctx)
	logger.Info("hello world", "user", "alice")

	// Add attributes to logger
	ctx = ctxlog.WithAttributes(ctx, "requestID", "123")
	logger = ctxlog.Logger(ctx)
	logger.Info("processing request")

	// Output will be JSON logs with context attributes
}

func ExampleNewJSONLogger() {
	// Create a new context with JSON logger
	ctx := context.Background()
	buf := &bytes.Buffer{}
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	ctx = ctxlog.NewJSONLogger(ctx, buf, opts)

	// Use the logger
	logger := ctxlog.Logger(ctx)
	logger.Debug("debug message")
	logger.Info("info message")

	// Output will be JSON formatted logs
}

func TestLogLevels(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	ctx = ctxlog.NewJSONLogger(ctx, buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// Test Info
	ctxlog.Info(ctx, "info message", "key", "value")
	if !bytes.Contains(buf.Bytes(), []byte("info message")) {
		t.Error("expected info message in log output")
	}
	buf.Reset()

	// Test Error
	ctxlog.Error(ctx, "error message", "err", "something failed")
	if !bytes.Contains(buf.Bytes(), []byte("error message")) {
		t.Error("expected error message in log output")
	}
	buf.Reset()

	// Test Debug
	ctxlog.Debug(ctx, "debug message", "debug", true)
	if !bytes.Contains(buf.Bytes(), []byte("debug message")) {
		t.Error("expected debug message in log output")
	}
	buf.Reset()

	// Test Warn
	ctxlog.Warn(ctx, "warning message", "warning", "careful")
	if !bytes.Contains(buf.Bytes(), []byte("warning message")) {
		t.Error("expected warning message in log output")
	}
	buf.Reset()

	// Test Log with custom level
	ctxlog.Log(ctx, slog.LevelInfo+1, "custom level", "level", "custom")
	if !bytes.Contains(buf.Bytes(), []byte("custom level")) {
		t.Error("expected custom level message in log output")
	}
}

func TestSource(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	ctx = ctxlog.NewJSONLogger(ctx, buf, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}

	check := func(line int, fn func()) {
		buf.Reset()
		fn()
		var logged struct {
			Source struct {
				File string `json:"file"`
				Line int    `json:"line"`
			} `json:"source"`
		}
		if err := json.Unmarshal(buf.Bytes(), &logged); err != nil {
			t.Fatalf("failed to unmarshal log output: %v\n%s", err, buf.String())
		}
		if got, want := logged.Source.File, file; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if got, want := logged.Source.Line, line; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	_, _, line, _ := runtime.Caller(0)
	check(line+1, func() { ctxlog.Info(ctx, "info") })
	_, _, line, _ = runtime.Caller(0)
	check(line+1, func() { ctxlog.Error(ctx, "error", nil) })
	_, _, line, _ = runtime.Caller(0)
	check(line+1, func() { ctxlog.Debug(ctx, "debug") })
	_, _, line, _ = runtime.Caller(0)
	check(line+1, func() { ctxlog.Warn(ctx, "warn") })

	logLogger := ctxlog.NewLogLogger(ctx, slog.LevelError)
	_, _, line, _ = runtime.Caller(0)
	check(line+1, func() { logLogger.Print("log logger") })
}
