// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/subcmd"
)

func TestLoggingConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	tests := []struct {
		name    string
		config  cmdutil.LoggingConfig
		wantErr bool
	}{
		{
			name: "defaults",
			config: cmdutil.LoggingConfig{
				Level:  0,
				Format: "text",
			},
			wantErr: false,
		},
		{
			name: "json-file",
			config: cmdutil.LoggingConfig{
				Level:  3,
				File:   logFile,
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "invalid-format",
			config: cmdutil.LoggingConfig{
				Format: "yaml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := tt.config.NewLogger()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("Config.NewLogger() returned nil logger")
			}
			if tt.config.File != "" && !tt.wantErr {
				if _, err := os.Stat(tt.config.File); os.IsNotExist(err) {
					t.Errorf("log file %q was not created", tt.config.File)
				}
			}
		})
	}
}

func TestLoggingFlagsRegister(t *testing.T) {
	fs := subcmd.NewFlagSet()
	var flags cmdutil.LoggingFlags
	if err := fs.RegisterFlagStruct(&flags, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func ExampleLoggingFlags() {
	// Typically these flags would be parsed from command line arguments.
	flags := cmdutil.LoggingFlags{
		Level:  2, // Info
		Format: "text",
	}
	cfg := flags.LoggingConfig()
	logger, err := cfg.NewLogger()
	if err != nil {
		panic(err)
	}
	slog.SetDefault(logger.Logger)
	slog.Info("hello world")
	// Output:
}

func TestLogBuildInfo(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "buildinfo.log")

	config := cmdutil.LoggingConfig{
		Level:  2, // Info level
		File:   logFile,
		Format: "json",
	}

	logger, err := config.NewLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Test the Logger.LogBuildInfo method
	logger.LogBuildInfo()

	// Test the standalone LogBuildInfo function
	cmdutil.LogBuildInfo(logger.Logger)

	if err := logger.Close(); err != nil {
		t.Fatalf("failed to close logger: %v", err)
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("log file is empty")
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var entries []map[string]any
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %s", line)
		}
		entries = append(entries, entry)
	}

	if got, want := len(entries), 2; got != want {
		t.Fatalf("expected %d log entries, got %d", want, got)
	}

	_, _, _, _, _, ok := cmdutil.VCSInfo()
	expectedMsg := "failed to determine version information"
	expectedLevel := "WARN"
	if ok {
		expectedMsg = "build info"
		expectedLevel = "INFO"
	}

	for i, entry := range entries {
		if msg, ok := entry["msg"].(string); !ok || msg != expectedMsg {
			t.Errorf("entry %d: unexpected message: got %v, want %q", i, entry["msg"], expectedMsg)
		}
		if level, ok := entry["level"].(string); !ok || level != expectedLevel {
			t.Errorf("entry %d: unexpected level: got %v, want %q", i, entry["level"], expectedLevel)
		}
	}
}

func TestLoggingToStdout(t *testing.T) {
	// Keep a backup of the real stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	// Restore functionality
	defer func() {
		os.Stdout = oldStdout
	}()

	config := cmdutil.LoggingConfig{
		Level:  2,
		File:   "-",
		Format: "text",
	}

	logger, err := config.NewLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("testing stdout logging")

	// Close the writer so we can read from the reader
	w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("testing stdout logging")) {
		t.Errorf("stdout logging failed, got: %q", output)
	}
}

func TestWithFlagOverrides(t *testing.T) {
	base := cmdutil.LoggingConfig{
		Level:      1,
		File:       "base.log",
		Format:     "text",
		SourceCode: false,
	}

	newFS := func(t *testing.T, args []string) (*flag.FlagSet, cmdutil.LoggingFlags) {
		t.Helper()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		var lf cmdutil.LoggingFlags
		fs.IntVar(&lf.Level, "log-level", 0, "")
		fs.StringVar(&lf.File, "log-file", "", "")
		fs.StringVar(&lf.Format, "log-format", "json", "")
		fs.BoolVar(&lf.SourceCode, "log-source-code", false, "")
		if err := fs.Parse(args); err != nil {
			t.Fatal(err)
		}
		return fs, lf
	}

	t.Run("no flags set leaves config unchanged", func(t *testing.T) {
		fs, lf := newFS(t, nil)
		if got := base.WithFlagOverrides(fs, lf); got != base {
			t.Errorf("got %+v, want %+v", got, base)
		}
	})

	t.Run("level overrides only level", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-level=3"})
		want := base
		want.Level = 3
		if got := base.WithFlagOverrides(fs, lf); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("file overrides only file", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-file=override.log"})
		want := base
		want.File = "override.log"
		if got := base.WithFlagOverrides(fs, lf); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("format overrides only format", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-format=json"})
		want := base
		want.Format = "json"
		if got := base.WithFlagOverrides(fs, lf); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("source-code overrides only source-code", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-source-code=true"})
		want := base
		want.SourceCode = true
		if got := base.WithFlagOverrides(fs, lf); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("all flags override all fields", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-level=3", "-log-file=all.log", "-log-format=json", "-log-source-code=true"})
		want := cmdutil.LoggingConfig{Level: 3, File: "all.log", Format: "json", SourceCode: true}
		if got := base.WithFlagOverrides(fs, lf); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestReplaceAttrNoTimeWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test-config.log")
	cfg := cmdutil.LoggingConfig{
		File:   logFile,
		Format: "json",
	}

	opts := &slog.HandlerOptions{
		ReplaceAttr: cmdutil.ReplaceAttrNoTime,
	}

	logger, err := cfg.NewLoggerOpts(opts)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	logger.Info("test message with config")
	logger.Close()

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	output := string(content)
	if strings.Contains(output, `"time":`) {
		t.Errorf("expected no time field, got: %s", output)
	}
	if !strings.Contains(output, "test message with config") {
		t.Errorf("expected log message, got: %s", output)
	}
}

// recordingWriteCloser captures everything written to it and records whether it
// was closed, so tests can assert both that log output is routed to a supplied
// writer and that the Logger takes ownership of closing it.
type recordingWriteCloser struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	closed int
}

func (w *recordingWriteCloser) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *recordingWriteCloser) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed++
	return nil
}

func (w *recordingWriteCloser) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *recordingWriteCloser) closeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

// TestWithUnderlyingWriteCloser verifies that a supplied WriteCloser receives
// the log output, for both constructors that accept logging options.
func TestWithUnderlyingWriteCloser(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func(cmdutil.LoggingConfig, ...cmdutil.LoggingOption) (*cmdutil.Logger, error)
	}{
		{"NewLogger", func(c cmdutil.LoggingConfig, opts ...cmdutil.LoggingOption) (*cmdutil.Logger, error) {
			return c.NewLogger(opts...)
		}},
		{"NewLoggerOpts", func(c cmdutil.LoggingConfig, opts ...cmdutil.LoggingOption) (*cmdutil.Logger, error) {
			return c.NewLoggerOpts(nil, opts...)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wc := &recordingWriteCloser{}
			cfg := cmdutil.LoggingConfig{Level: 2, Format: "text"}

			logger, err := tc.new(cfg, cmdutil.WithWriteCloser(wc))
			if err != nil {
				t.Fatalf("creating logger: %v", err)
			}
			logger.Info("hello", "key", "value")

			if got := wc.String(); !strings.Contains(got, "hello") || !strings.Contains(got, "key=value") {
				t.Errorf("output %q does not contain the logged message", got)
			}
		})
	}
}

// TestWithWriteCloserOverridesFile verifies that a supplied
// WriteCloser is used in preference to the configured File, which is left
// untouched.
func TestWithWriteCloserOverridesFile(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "unused.log")
	wc := &recordingWriteCloser{}
	cfg := cmdutil.LoggingConfig{Level: 2, Format: "text", File: logFile}

	logger, err := cfg.NewLogger(cmdutil.WithWriteCloser(wc))
	if err != nil {
		t.Fatalf("creating logger: %v", err)
	}
	logger.Info("to the writer")

	if got := wc.String(); !strings.Contains(got, "to the writer") {
		t.Errorf("output %q does not contain the logged message", got)
	}
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Errorf("the configured log file %v was created despite a writer being supplied (err %v)", logFile, err)
	}
}

// TestWithWriteCloserClose verifies that Logger.Close closes the
// supplied WriteCloser: the Logger takes ownership of it, as it does of a log
// file it opened itself.
func TestWithWriteCloserClose(t *testing.T) {
	wc := &recordingWriteCloser{}
	cfg := cmdutil.LoggingConfig{Level: 2, Format: "text"}

	logger, err := cfg.NewLogger(cmdutil.WithWriteCloser(wc))
	if err != nil {
		t.Fatalf("creating logger: %v", err)
	}
	if got := wc.closeCount(); got != 0 {
		t.Errorf("the writer was closed %d times before Logger.Close", got)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if got, want := wc.closeCount(), 1; got != want {
		t.Errorf("the writer was closed %d times, want %d", got, want)
	}
}

// TestWithWriteCloserFormat verifies that the configured format is
// applied to a supplied writer, ie. the option changes only the destination.
func TestWithWriteCloserFormat(t *testing.T) {
	wc := &recordingWriteCloser{}
	cfg := cmdutil.LoggingConfig{Level: 2, Format: "json"}

	logger, err := cfg.NewLogger(cmdutil.WithWriteCloser(wc))
	if err != nil {
		t.Fatalf("creating logger: %v", err)
	}
	logger.Info("structured", "key", "value")
	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var rec map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(wc.String())), &rec); err != nil {
		t.Fatalf("output %q is not JSON: %v", wc.String(), err)
	}
	if got, want := rec["msg"], "structured"; got != want {
		t.Errorf("msg: got %v, want %v", got, want)
	}
	if got, want := rec["key"], "value"; got != want {
		t.Errorf("key: got %v, want %v", got, want)
	}
}

// TestNoUnderlyingWriteCloser verifies that omitting the option leaves the
// file-based behaviour, and its no-op closer, unchanged.
func TestNoUnderlyingWriteCloser(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "test.log")
	cfg := cmdutil.LoggingConfig{Level: 2, Format: "text", File: logFile}

	logger, err := cfg.NewLogger()
	if err != nil {
		t.Fatalf("creating logger: %v", err)
	}
	logger.Info("to the file")
	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading %v: %v", logFile, err)
	}
	if !strings.Contains(string(data), "to the file") {
		t.Errorf("log file %q does not contain the logged message", data)
	}

	// Stderr, the default, is not owned by the Logger, so Close is a no-op.
	stderrLogger, err := cmdutil.LoggingConfig{Level: 2, Format: "text"}.NewLogger()
	if err != nil {
		t.Fatalf("creating logger: %v", err)
	}
	if err := stderrLogger.Close(); err != nil {
		t.Errorf("Close on a stderr logger: %v", err)
	}
}
