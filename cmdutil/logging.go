// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// IsExplicitlySet returns true if the named flag was explicitly provided on
// the command line (i.e. after FlagSet.Parse was called). It relies on
// flag.FlagSet.Visit, which only visits flags that were set during parsing.
func IsExplicitlySet(fs *flag.FlagSet, name string) bool {
	if fs == nil {
		return false
	}
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// LoggingFlags represents common logging related command line flags.
type LoggingFlags struct {
	Level      int    `subcmd:"log-level,0,'logging level: 0=error, 1=warn, 2=info, 3=debug'"`
	File       string `subcmd:"log-file,,'log file path. If not specified logs are written to stderr, if set to - logs are written to stdout'"`
	TempFile   string `subcmd:"log-tempfile,,'create a unique log file from this os.CreateTemp pattern, eg. myapp-*.log, in os.TempDir unless the pattern includes a directory, which is created if needed. Mutually exclusive with log-file'"`
	Format     string `subcmd:"log-format,json,'log format: text or json'"`
	SourceCode bool   `subcmd:"log-source-code,false,'include source code file and line number in logs'"`
}

// LoggingConfig represents a logging configuration.
type LoggingConfig struct {
	Level      int    `yaml:"level" doc:"logging level: 0=error, 1=warn, 2=info, 3=debug"`
	File       string `yaml:"file" doc:"log file path. If not specified logs are written to stderr, if set to - logs are written to stdout."`
	TempFile   string `yaml:"temp_file" doc:"create a unique log file from this os.CreateTemp pattern, eg. myapp-*.log, in os.TempDir unless the pattern includes a directory, which is created if needed. Mutually exclusive with file."`
	Format     string `yaml:"format" doc:"log format: text or json"`
	SourceCode bool   `yaml:"source_code" doc:"include source code file and line number in logs"`
}

// LoggingConfig returns the logging configuration represented by the flags.
func (lf LoggingFlags) LoggingConfig() LoggingConfig {
	return LoggingConfig(lf)
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
	f    io.Closer
	path string
}

// Close closes the log file, if any.
func (l *Logger) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

// Path returns the name of the log file, including the generated name for
// LoggingConfig.TempFile. It is empty if logging to stderr, stdout or a
// writer specified via WithWriteCloser.
func (l *Logger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// LogBuildInfo logs build information using the logger.
func (l *Logger) LogBuildInfo() {
	LogBuildInfo(l.Logger)
}

type noopCloser struct{}

func (noopCloser) Close() error {
	return nil
}

// Leveler returns the slog.Leveler for the configured level.
func (c LoggingConfig) Leveler() slog.Leveler {
	return leveler{level: c.Level}
}

// Options returns the slog.HandlerOptions for the configuration.
func (c LoggingConfig) Options() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		AddSource: c.SourceCode,
		Level:     leveler{level: c.Level},
	}
}

// LoggingOption represents an option for configuring a logger
// beyond the slog.HandlerOptions provided by the LoggingConfig.
type LoggingOption func(*loggingOptions)

// WithWriteCloser sets the io.Writer for the slog.Logger to use and the Closer
// to return. It is intended to allow for setting a writer that supports log
// rotation or other logging destinations. If set, the File and TempFile fields of the
// LoggingConfig are ignored and the WriteCloser passed here is used instead.
func WithWriteCloser(wr io.WriteCloser) LoggingOption {
	return func(o *loggingOptions) {
		o.wr = wr
	}
}

type loggingOptions struct {
	wr io.WriteCloser
}

// NewLogger creates a new logger based on the configuration.
func (c LoggingConfig) NewLogger(opts ...LoggingOption) (*Logger, error) {
	lo := loggingOptions{}
	for _, opt := range opts {
		opt(&lo)
	}
	return c.newLogger(lo, c.Options())
}

// WithFlagOverrides returns a new LoggingConfig with fields overridden by
// the explicitly set flags in the provided FlagSet.
func (c LoggingConfig) WithFlagOverrides(fs *flag.FlagSet, lf LoggingFlags) LoggingConfig {
	if fs == nil {
		return c
	}
	var sawFile, sawTempFile bool
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "log-level":
			c.Level = lf.Level
		case "log-file":
			sawFile = true
			c.File = lf.File
		case "log-tempfile":
			sawTempFile = true
			c.TempFile = lf.TempFile
		case "log-format":
			c.Format = lf.Format
		case "log-source-code":
			c.SourceCode = lf.SourceCode
		}
	})
	if sawFile && !sawTempFile {
		c.TempFile = ""
	} else if sawTempFile && !sawFile {
		c.File = ""
	}
	return c
}

func (c LoggingConfig) newLogger(lo loggingOptions, opts *slog.HandlerOptions) (*Logger, error) {
	var handler slog.Handler
	var closer io.Closer
	var out io.Writer
	var path string
	switch c.Format {
	case "json", "text", "":
	default:
		return nil, fmt.Errorf("unknown log format %q", c.Format)
	}
	if lo.wr == nil {
		if c.File != "" && c.TempFile != "" {
			return nil, fmt.Errorf("log file %q and temporary log file %q are mutually exclusive", c.File, c.TempFile)
		}
		switch {
		case c.TempFile != "":
			dir, pattern := filepath.Split(c.TempFile)
			if pattern == "" {
				return nil, fmt.Errorf("invalid temporary log file pattern %q: missing filename pattern", c.TempFile)
			}
			if dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return nil, fmt.Errorf("failed to create directory for temporary log file %q: %w", c.TempFile, err)
				}
			}
			f, err := os.CreateTemp(dir, pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to create temporary log file %q: %w", c.TempFile, err)
			}
			closer, out, path = f, f, f.Name()
		case c.File == "":
			out = os.Stderr
			closer = &noopCloser{}
		case c.File == "-":
			out = os.Stdout
			closer = &noopCloser{}
		default:
			f, err := os.OpenFile(c.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return nil, fmt.Errorf("failed to open log file %q: %w", c.File, err)
			}
			closer, out, path = f, f, c.File
		}
	} else {
		out = lo.wr
		closer = lo.wr
	}
	switch c.Format {
	case "json":
		handler = slog.NewJSONHandler(out, opts)
	case "text", "":
		handler = slog.NewTextHandler(out, opts)
	}
	return &Logger{Logger: slog.New(handler), f: closer, path: path}, nil
}

// NewLoggerOpts creates a new logger based on the configuration and custom
// handler options.
func (c LoggingConfig) NewLoggerOpts(handlerOpts *slog.HandlerOptions, loggingOpts ...LoggingOption) (*Logger, error) {
	if handlerOpts == nil {
		handlerOpts = c.Options()
	}
	lo := loggingOptions{}
	for _, opt := range loggingOpts {
		opt(&lo)
	}
	return c.newLogger(lo, handlerOpts)
}

// NewLoggerMust is like NewLogger but panics on error.
func (c LoggingConfig) NewLoggerMust(opts *slog.HandlerOptions, loggingOpts ...LoggingOption) *Logger {
	logger, err := c.NewLoggerOpts(opts, loggingOpts...)
	if err != nil {
		panic(err)
	}
	return logger
}

// LogBuildInfo logs build information using the provided logger.
func LogBuildInfo(logger *slog.Logger) {
	goVersion, version, when, modTime, dirty, ok := VCSInfo()
	if !ok {
		logger.Warn("failed to determine version information")
		return
	}
	logger.Info("build info", "go.version", goVersion, "commit", version, "commit.date", when, "exe.modtime", modTime, "dirty", dirty)
}

// ReplaceAttrNoTime returns a slog.Attr with the time attribute removed.
// This is useful for tests where the time is not deterministic.
func ReplaceAttrNoTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}
