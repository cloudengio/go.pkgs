// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/structdoc"
	"gopkg.in/yaml.v3"
)

// ConfigCommands implements commands to show, and to document, the YAML
// configuration of type T. The methods are subcmd.Runners; see
// ConfigExtension for adding them to a command set.
type ConfigCommands[T any] struct {
	// Summary is written at the top of the output of Describe.
	Summary string
	// Describers are called by Describe, in order, after the documentation
	// for T, to write further documentation to the output.
	Describers []func(io.Writer) error
	// Output is where the output is written, os.Stdout if nil.
	Output io.Writer
}

func (c ConfigCommands[T]) output() io.Writer {
	if c.Output == nil {
		return os.Stdout
	}
	return c.Output
}

// Show writes the configuration of type T carried by ctx, using
// cmdutil.ContextWithValue, as YAML preceded by comments naming the files
// that it was read from if the context carries the *Located
// for them.
func (c ConfigCommands[T]) Show(ctx context.Context, _ any, _ []string) error {
	cfg, ok := cmdutil.ContextValue[T](ctx)
	if !ok {
		return fmt.Errorf("no configuration of type %T in the context", cfg)
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error formatting the configuration as yaml: %w", err)
	}
	w := c.output()
	if loc, ok := cmdutil.ContextValue[*Located](ctx); ok && loc != nil {
		if len(loc.YAML) == 0 {
			fmt.Fprintln(w, "# Configuration file: none")
		}
		for _, f := range loc.YAML {
			fmt.Fprintf(w, "# Configuration file: %s\n", f)
		}
		for _, f := range loc.Env {
			fmt.Fprintf(w, "# Environment variables file: %s\n", f)
		}
	}
	_, err = w.Write(out)
	return err
}

// Describe writes the documentation for the configuration type T, from its
// 'doc' struct tags, see structdoc.Describe, followed by whatever Describers
// write.
func (c ConfigCommands[T]) Describe(_ context.Context, _ any, _ []string) error {
	var cfg T
	desc, err := structdoc.Describe(cfg, "doc", c.Summary)
	if err != nil {
		return fmt.Errorf("error describing the configuration: %w", err)
	}
	w := c.output()
	fmt.Fprintln(w, desc)
	for _, d := range c.Describers {
		if err := d(w); err != nil {
			return err
		}
	}
	return nil
}
