// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
	"context"
	"fmt"
	"strings"

	"cloudeng.io/cmdutil/subcmd"
)

// ExampleNewExtension shows how to inject subcommands into a command tree at
// runtime. The base YAML spec uses a Go template directive to mark the
// insertion point; MustFromYAMLTemplate expands the template with the
// extension's YAML fragment, and MustAddExtensions wires in the runners.
//
// This pattern lets library packages provide self-contained command groups
// (YAML spec + runners) that host CLIs can embed without knowing the details.
func ExampleNewExtension() {
	// The base spec marks where the extension's subcommands will appear.
	const baseSpec = `name: tool
summary: a CLI tool
commands:
  - name: status
    summary: show tool status
  {{range subcmdExtension "db"}}
  {{.}}{{end}}
`

	// The extension supplies the YAML fragment for its own subcommands.
	const dbYAML = `
- name: db
  summary: database commands
  commands:
    - name: migrate
      summary: run pending migrations
    - name: reset
      summary: reset the database to an empty state
`

	type dbFlags struct {
		DryRun bool `subcmd:"dry-run,false,print what would happen without doing it"`
	}

	dbExt := subcmd.NewExtension("db", dbYAML,
		func(cmd *subcmd.CommandSetYAML) error {
			// The context carries the command's FlagSet (see
			// subcmd.FlagSetFromContext) but not the command's name,
			// so capture the name in a closure when registering the runner.
			run := func(name string) func(context.Context, any, []string) error {
				return func(_ context.Context, f any, _ []string) error {
					fl := f.(*dbFlags)
					if fl.DryRun {
						fmt.Printf("dry-run: %s\n", name)
					} else {
						fmt.Printf("running: %s\n", name)
					}
					return nil
				}
			}
			cmd.Set("db", "migrate").MustRunner(run("migrate"), &dbFlags{})
			cmd.Set("db", "reset").MustRunner(run("reset"), &dbFlags{})
			return nil
		},
	)

	// MustFromYAMLTemplate expands the template and builds the command tree.
	// The extension's YAML is injected at the {{range subcmdExtension "db"}} site.
	cmd := subcmd.MustFromYAMLTemplate(baseSpec, dbExt)

	// MustAddExtensions calls each extension's Set function to register runners.
	cmd.MustAddExtensions()

	// The command tree now contains both "status" and the "db" subtree.
	fmt.Println(strings.Contains(cmd.String(), "db"))

	// Dispatching runs the registered runner for the named subcommand.
	ctx := context.Background()
	if err := cmd.DispatchWithArgs(ctx, "tool", "db", "migrate"); err != nil {
		panic(err)
	}
	if err := cmd.DispatchWithArgs(ctx, "tool", "db", "reset", "--dry-run"); err != nil {
		panic(err)
	}
	// Output:
	// true
	// running: migrate
	// dry-run: reset
}

// ExampleMergeExtensions shows how to combine multiple independent extensions
// under a single name so they appear at the same insertion point in the spec.
func ExampleMergeExtensions() {
	const baseSpec = `name: tool
summary: a CLI tool
commands:
  {{range subcmdExtension "extras"}}
  {{.}}{{end}}
`

	type noFlags struct{}

	makeExt := func(name string) subcmd.Extension {
		yaml := fmt.Sprintf("- name: %s\n  summary: %s command\n", name, name)
		return subcmd.NewExtension(name, yaml,
			func(cmd *subcmd.CommandSetYAML) error {
				cmd.Set(name).MustRunner(
					func(_ context.Context, _ any, _ []string) error {
						fmt.Printf("%s ran\n", name)
						return nil
					}, &noFlags{},
				)
				return nil
			},
		)
	}

	alpha := makeExt("alpha")
	beta := makeExt("beta")

	// MergeExtensions combines alpha and beta under the single name "extras",
	// so both are injected at the {{range subcmdExtension "extras"}} site.
	merged := subcmd.MergeExtensions("extras", alpha, beta)

	cmd := subcmd.MustFromYAMLTemplate(baseSpec, merged)
	cmd.MustAddExtensions()

	fmt.Println(strings.Contains(cmd.String(), "alpha"))
	fmt.Println(strings.Contains(cmd.String(), "beta"))
	// Output:
	// true
	// true
}
