// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package cmdexec provides a means of executing multiple subcommands
// with the ability to expand the command line arguments using Go's
// text/template package and environment variables.
package cmdexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// Runner represents a command Runner
type Runner struct {
	name string
	opts options
}

type options struct {
	dryRun         bool
	verbose        bool
	workingDir     string
	prefix         []string
	stdout, stderr io.Writer
	templateVars   any
	templateFuncs  template.FuncMap
	logger         func(string, ...any) (int, error)
	mapping        func(string) string
}

// Option represents an option to New.
type Option func(*options)

// WithDryRun logs the commands that would be executed but does not
// actually execute them.
func WithDryRun(v bool) Option {
	return func(o *options) {
		o.dryRun = v
	}
}

// WithVerbose sets the verbose flag for the commands which
// generally results in the expanded command line execeuted being
// logged.
func WithVerbose(v bool) Option {
	return func(o *options) {
		o.verbose = v
	}
}

// WithWorkingDir sets the working directory for the commands.
func WithWorkingDir(v string) Option {
	return func(o *options) {
		o.workingDir = v
	}
}

// WithCommandsPrefix sets a common set of arguments prepended to all
// commands.
func WithCommandsPrefix(v ...string) Option {
	return func(o *options) {
		o.prefix = v
	}
}

// WithStdout sets the writer to which the standard output of the
// commands will be written.
func WithStdout(v io.Writer) Option {
	return func(o *options) {
		o.stdout = v
	}
}

// WithStderr sets the writer to which the standard error of the
// commands will be written.
func WithStderr(v io.Writer) Option {
	return func(o *options) {
		o.stderr = v
	}
}

// WithLogger sets the logger function to be used to log the verbose
// and dry run output. The default is fmt.Printf.
func WithLogger(v func(string, ...any) (int, error)) Option {
	return func(o *options) {
		o.logger = v
	}
}

// WithExpandMapping sets the mapping function to be used to expand
// environment variables in the command line arguments. The default
// is os.Getenv.
func WithExpandMapping(v func(string) string) Option {
	return func(o *options) {
		o.mapping = v
	}
}

// WithTemplateFuncs sets the template functions to be used to expand
// the command line arguments.
func WithTemplateFuncs(v template.FuncMap) Option {
	return func(o *options) {
		o.templateFuncs = v
	}
}

// WithTemplateVars sets the template variables to be used to expand
// the command line arguments.
func WithTemplateVars(v any) Option {
	return func(o *options) {
		o.templateVars = v
	}
}

// New creates a new Runner instance with the supplied name and options.
func New(name string, opts ...Option) *Runner {
	r := &Runner{name: name}
	r.opts.logger = fmt.Printf
	r.opts.stderr = os.Stderr
	r.opts.stdout = os.Stdout
	r.opts.mapping = os.Getenv
	for _, o := range opts {
		o(&r.opts)
	}
	return r
}

// ExpandCommandLine expands the supplied command line arguments using
// the supplied template functions and variables. Template expansion
// is performed before variable expansion.
func (r *Runner) ExpandCommandLine(args ...string) ([]string, error) {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		tpl, err := template.New(r.name).Funcs(r.opts.templateFuncs).Parse(arg)
		if err != nil {
			return nil, err
		}
		var out strings.Builder
		if err := tpl.Execute(&out, r.opts.templateVars); err != nil {
			return nil, err
		}
		e := os.Expand(out.String(), r.opts.mapping)
		expanded = append(expanded, e)
	}
	return expanded, nil
}

// Run executes the supplied commands with template expansion using
// the supplied template functions and variables.
func (r *Runner) Run(ctx context.Context, cmds ...string) error {
	if len(cmds) == 0 {
		return fmt.Errorf("no commands for %v", r.name)
	}
	all := append([]string{}, r.opts.prefix...)
	all = append(all, cmds...)
	expanded, err := r.ExpandCommandLine(all...)
	if err != nil {
		return err
	}
	return r.runInDir(ctx, expanded)
}

func (r *Runner) runInDir(ctx context.Context, args []string) error {
	binary := args[0]
	if len(args) == 1 {
		args = []string{}
	} else {
		args = args[1:]
	}
	if r.opts.dryRun || r.opts.verbose {
		r.opts.logger("[%v]: %v %v\n", r.opts.workingDir, binary, strings.Join(args, " "))
	}
	if r.opts.dryRun {
		return nil
	}
	fmt.Printf(">>>> %v %v\n", binary, args)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = r.opts.workingDir
	cmd.Stdout = r.opts.stdout
	cmd.Stderr = r.opts.stderr
	return cmd.Run()
}
