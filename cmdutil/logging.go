// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"fmt"
	"log/slog"
	"os"
)

// LoggingFlags represents common logging related command line flags.
type LoggingFlags struct {
	Level      int    `subcmd:"log-level,0,'logging level: 0=error, 1=warn, 2=info, 3=debug'"`
	File       string `subcmd:"log-file,,'log file path. If not specified logs are written to stderr.'"`
	Format     string `subcmd:"log-format,json,'log format: text or json'"`
	SourceCode bool   `subcmd:"log-source-code,false,'include source code file and line number in logs'"`
}

// LoggingConfig represents logging Loggingconfiguration.
type LoggingConfig struct {
	Level      int
	File       string
	Format     string
	SourceCode bool
}

// LoggingConfig returns the logging configuration represented by the flags.
func (lf *LoggingFlags) LoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:      lf.Level,
		File:       lf.File,
		Format:     lf.Format,
		SourceCode: lf.SourceCode,
	}
}

type leveler struct {
	level int
}

func (l leveler) Level() slog.Level {
	switch {
	case l.level <= 0:
		return slog.LevelError
	case l.level == 1:
		return slog.LevelWarn
	case l.level == 2:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

// NewLogger creates a new logger based on the configuration.
func (c LoggingConfig) NewLogger() (*slog.Logger, error) {
	opts := &slog.HandlerOptions{
		AddSource: c.SourceCode,
		Level:     leveler{level: c.Level},
	}
	var handler slog.Handler
	out := os.Stderr
	if c.File != "" {
		f, err := os.OpenFile(c.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %q: %w", c.File, err)
		}
		out = f
	}
	switch c.Format {
	case "json":
		handler = slog.NewJSONHandler(out, opts)
	case "text", "":
		handler = slog.NewTextHandler(out, opts)
	default:
		return nil, fmt.Errorf("unknown log format %q", c.Format)
	}
	return slog.New(handler), nil
}

// NewLoggerMust is like NewLogger but panics on error.
func (c LoggingConfig) NewLoggerMust() *slog.Logger {
	logger, err := c.NewLogger()
	if err != nil {
		panic(err)
	}
	return logger
}
