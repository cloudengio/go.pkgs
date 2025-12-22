// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// LoggingFlags represents common logging related command line flags.
type LoggingFlags struct {
	Level      int    `subcmd:"log-level,0,'logging level: 0=error, 1=warn, 2=info, 3=debug'"`
	File       string `subcmd:"log-file,,'log file path. If not specified logs are written to stderr, if set to - logs are written to stdout'"`
	Format     string `subcmd:"log-format,json,'log format: text or json'"`
	SourceCode bool   `subcmd:"log-source-code,false,'include source code file and line number in logs'"`
}

// LoggingConfig represents a logging configuration.
type LoggingConfig struct {
	Level      int    `yaml:"level" cmd:"logging level: 0=error, 1=warn, 2=info, 3=debug"`
	File       string `yaml:"file" cmd:"log file path. If not specified logs are written to stderr."`
	Format     string `yaml:"format" cmd:"log format: text or json"`
	SourceCode bool   `yaml:"source_code" cmd:"include source code file and line number in logs"`
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

// Logger represents a logger with an optional closer for the log file
// if one is specified.
type Logger struct {
	*slog.Logger
	f io.Closer
}

func (l *Logger) Close() error {
	return l.f.Close()
}

// LogBuildInfo logs build information using the logger.
func (l *Logger) LogBuildInfo() {
	LogBuildInfo(l.Logger)
}

type noopCloser struct{}

func (noopCloser) Close() error {
	return nil
}

// NewLogger creates a new logger based on the configuration.
func (c LoggingConfig) NewLogger() (*Logger, error) {
	opts := &slog.HandlerOptions{
		AddSource: c.SourceCode,
		Level:     leveler{level: c.Level},
	}
	var handler slog.Handler
	var closer io.Closer
	var out io.Writer
	switch c.File {
	case "":
		out = os.Stderr
		closer = &noopCloser{}
	case "-":
		out = os.Stdout
		closer = &noopCloser{}
	default:
		f, err := os.OpenFile(c.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %q: %w", c.File, err)
		}
		closer = f
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
	return &Logger{Logger: slog.New(handler), f: closer}, nil
}

// NewLoggerMust is like NewLogger but panics on error.
func (c LoggingConfig) NewLoggerMust() *Logger {
	logger, err := c.NewLogger()
	if err != nil {
		panic(err)
	}
	return logger
}

// LogBuildInfo logs build information using the provided logger.
func LogBuildInfo(logger *slog.Logger) {
	goVersion, version, when, dirty, ok := VCSInfo()
	if !ok {
		logger.Warn("failed to determine version information")
		return
	}
	logger.Info("build info", "go.version", goVersion, "commit", version, "build.date", when, "dirty", dirty)
}
