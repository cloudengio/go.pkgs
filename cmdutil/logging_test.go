// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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
	var entries []map[string]interface{}
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %s", line)
		}
		entries = append(entries, entry)
	}

	if got, want := len(entries), 2; got != want {
		t.Fatalf("expected %d log entries, got %d", want, got)
	}

	_, _, _, _, ok := cmdutil.VCSInfo()
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
		w.Close()
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
