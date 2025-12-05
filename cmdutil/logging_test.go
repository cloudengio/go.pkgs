// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil_test

import (
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
