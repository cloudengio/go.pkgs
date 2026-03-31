// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ctxlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"testing"

	"cloudeng.io/logging/ctxlog"
)

// saveLogState / restoreLogState bracket tests that mutate the global log
// package so that they don't interfere with each other or with other packages.
func saveLogState() (flags int, output io.Writer) {
	return log.Flags(), log.Writer()
}

func restoreLogState(flags int, output io.Writer) {
	log.SetFlags(flags)
	log.SetOutput(output)
}

func TestCaptureLog(t *testing.T) {
	flags, output := saveLogState()
	defer restoreLogState(flags, output)

	buf := &bytes.Buffer{}
	ctx := ctxlog.NewJSONLogger(context.Background(), buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctxlog.CaptureLog(ctx, slog.LevelInfo)

	log.Print("hello from log")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if got := entry["msg"]; got != "hello from log" {
		t.Errorf("msg: got %q, want %q", got, "hello from log")
	}
	if got := entry["level"]; got != "INFO" {
		t.Errorf("level: got %q, want %q", got, "INFO")
	}
}

func TestCaptureLogLevel(t *testing.T) {
	flags, output := saveLogState()
	defer restoreLogState(flags, output)

	buf := &bytes.Buffer{}
	ctx := ctxlog.NewJSONLogger(context.Background(), buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctxlog.CaptureLog(ctx, slog.LevelWarn)

	log.Print("a warning")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if got := entry["level"]; got != "WARN" {
		t.Errorf("level: got %q, want %q", got, "WARN")
	}
}

// TestCaptureLogNoTimestampPrefix verifies that log's own date/time prefix
// is not prepended to the message (CaptureLog clears log.Flags).
func TestCaptureLogNoTimestampPrefix(t *testing.T) {
	flags, output := saveLogState()
	defer restoreLogState(flags, output)

	buf := &bytes.Buffer{}
	ctx := ctxlog.NewJSONLogger(context.Background(), buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	ctxlog.CaptureLog(ctx, slog.LevelInfo)

	log.Print("clean message")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	// If log's flags were not cleared, the message would start with a date/time
	// prefix like "2009/11/10 " and the msg field would not match exactly.
	if got := entry["msg"]; got != "clean message" {
		t.Errorf("msg: got %q, want %q (log flags may not have been cleared)", got, "clean message")
	}
}

// TestCaptureLogBelowLevel verifies that messages are suppressed when the
// slog handler's minimum level is above the level passed to CaptureLog.
func TestCaptureLogBelowLevel(t *testing.T) {
	flags, output := saveLogState()
	defer restoreLogState(flags, output)

	buf := &bytes.Buffer{}
	// Handler only accepts Warn and above.
	ctx := ctxlog.NewJSONLogger(context.Background(), buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})

	// CaptureLog at Info — below the handler's threshold.
	ctxlog.CaptureLog(ctx, slog.LevelInfo)

	log.Print("should be dropped")

	if buf.Len() != 0 {
		t.Errorf("expected no output, got: %s", buf.String())
	}
}

// TestCaptureLogPrintf verifies that formatted messages arrive intact.
func TestCaptureLogPrintf(t *testing.T) {
	flags, output := saveLogState()
	defer restoreLogState(flags, output)

	buf := &bytes.Buffer{}
	ctx := ctxlog.NewJSONLogger(context.Background(), buf, nil)

	ctxlog.CaptureLog(ctx, slog.LevelInfo)

	log.Printf("value=%d", 42)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if got := entry["msg"]; got != "value=42" {
		t.Errorf("msg: got %q, want %q", got, "value=42")
	}
}
