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
	if p := logger.Path(); p != "" {
		t.Errorf("expected empty path for stdout logger, got %q", p)
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
		fs.StringVar(&lf.TempFile, "log-tempfile", "", "")
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

	t.Run("tempfile replaces file", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-tempfile=x-*.log"})
		want := base
		want.File, want.TempFile = "", "x-*.log"
		if got := base.WithFlagOverrides(fs, lf); got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("file replaces tempfile", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-file=f.log"})
		cfg := cmdutil.LoggingConfig{TempFile: "x-*.log"}
		want := cmdutil.LoggingConfig{File: "f.log"}
		if got := cfg.WithFlagOverrides(fs, lf); got != want {
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

	t.Run("setting both file and tempfile preserves both for validation", func(t *testing.T) {
		fs, lf := newFS(t, []string{"-log-file=a.log", "-log-tempfile=b-*.log"})
		got := base.WithFlagOverrides(fs, lf)
		if got.File != "a.log" || got.TempFile != "b-*.log" {
			t.Fatalf("got %+v, want both File and TempFile set", got)
		}
		if _, err := got.NewLogger(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Errorf("expected mutually exclusive error, got %v", err)
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
	if p := logger.Path(); p != "" {
		t.Errorf("expected empty path when using WithWriteCloser, got %q", p)
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
	if p := logger.Path(); p != logFile {
		t.Errorf("Path(): got %q, want %q", p, logFile)
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
	if p := stderrLogger.Path(); p != "" {
		t.Errorf("expected empty path for stderr logger, got %q", p)
	}
	if err := stderrLogger.Close(); err != nil {
		t.Errorf("Close on a stderr logger: %v", err)
	}
}

func TestLoggingTempFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir) // unix
	t.Setenv("TMP", dir)    // windows
	t.Setenv("TEMP", dir)   // windows

	for _, pattern := range []string{"app-*.log", filepath.Join(dir, "sub-*.log")} {
		cfg := cmdutil.LoggingConfig{Level: 2, Format: "text", TempFile: pattern}
		l1, err := cfg.NewLogger()
		if err != nil {
			t.Fatal(err)
		}
		l2, err := cfg.NewLogger()
		if err != nil {
			t.Fatal(err)
		}
		if l1.Path() == "" || l1.Path() == l2.Path() {
			t.Errorf("%v: paths not unique: %q %q", pattern, l1.Path(), l2.Path())
		}
		got, _ := filepath.EvalSymlinks(filepath.Dir(l1.Path()))
		want, _ := filepath.EvalSymlinks(dir)
		if got != want {
			t.Errorf("%v: got %v, want %v", pattern, got, want)
		}
		l1.Info("hello temp")
		if err := l1.Close(); err != nil {
			t.Fatal(err)
		}
		if err := l2.Close(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(l1.Path())
		if err != nil || !strings.Contains(string(data), "hello temp") {
			t.Errorf("%v: %q: %v", pattern, data, err)
		}
	}
}

func TestLoggingTempFileErrors(t *testing.T) {
	cfg := cmdutil.LoggingConfig{File: "a.log", TempFile: "b-*.log"}
	if _, err := cfg.NewLogger(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("unexpected error: %v", err)
	}
	cfg = cmdutil.LoggingConfig{TempFile: t.TempDir() + string(filepath.Separator)}
	if _, err := cfg.NewLogger(); err == nil || !strings.Contains(err.Error(), "missing filename pattern") {
		t.Errorf("expected error for trailing separator pattern, got: %v", err)
	}
	// A missing directory is created.
	cfg = cmdutil.LoggingConfig{TempFile: filepath.Join(t.TempDir(), "missing", "b-*.log")}
	l, err := cfg.NewLogger()
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.Path()); err != nil {
		t.Error(err)
	}
	// But not if it can't be, eg. its parent is a file.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg = cmdutil.LoggingConfig{TempFile: filepath.Join(file, "sub", "b-*.log")}
	if _, err := cfg.NewLogger(); err == nil {
		t.Error("expected an error when the directory cannot be created")
	}
	// A failed logger must not leave a temp file behind.
	dir := t.TempDir()
	cfg = cmdutil.LoggingConfig{TempFile: filepath.Join(dir, "b-*.log"), Format: "bad"}
	if _, err := cfg.NewLogger(); err == nil {
		t.Error("expected an error for a bad format")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("bad format left %d file(s)", len(entries))
	}
	var nl *cmdutil.Logger
	if err := nl.Close(); err != nil {
		t.Error(err)
	}
	if p := nl.Path(); p != "" {
		t.Errorf("got %q", p)
	}
	if p := (&cmdutil.Logger{}).Path(); p != "" {
		t.Errorf("got %q", p)
	}
}
