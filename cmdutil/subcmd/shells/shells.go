// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package shells generates bash and zsh shell completion scripts from a
// cloudeng.io/cmdutil/subcmd command tree.
//
// Users of subcmd can wire a completions subcommand directly into a YAML
// command spec using Runner and Flags:
//
//	cmdSet.Set("completions").MustRunner(
//	    shells.Runner(cmdSet.CommandSet, nil),
//	    &shells.Flags{},
//	)
//
// Or construct one programmatically with NewCommand.
package shells

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"cloudeng.io/cmdutil/subcmd"
)

const (
	// Bash is the shell name for bash completion scripts.
	BashShell = "bash"
	// ZshShell is the shell name for zsh completion scripts.
	ZshShell = "zsh"
)

// SupportedShells returns the names of the shells supported by this package.
func SupportedShells() []string {
	return []string{BashShell, ZshShell}
}

// Flags defines the flags accepted by the subcommand runner returned by Runner.
type Flags struct {
	Program string `subcmd:"program,,program name to generate completions for (defaults to the executable name)"`
}

// Runner returns a subcmd.Runner that writes a completion script for the shell
// named in args[0] ("bash" or "zsh") to out (or os.Stdout if out is nil).
// The command should accept one argument (<shell>) and use &Flags{} as its
// flag struct.
func Runner(cmds *subcmd.CommandSet, out io.Writer) subcmd.Runner {
	if out == nil {
		out = os.Stdout
	}
	return func(_ context.Context, flagValues any, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("expected exactly one shell argument (%s)", strings.Join(SupportedShells(), ", "))
		}
		prog := ""
		if f, ok := flagValues.(*Flags); ok && f != nil {
			prog = f.Program
		}
		return Generate(out, args[0], prog, cmds)
	}
}

// NewCommand constructs a subcmd.Command named name that generates shell
// completions for cmds and writes them to out (or os.Stdout if out is nil).
func NewCommand(name string, cmds *subcmd.CommandSet, out io.Writer) *subcmd.Command {
	fs := subcmd.MustRegisteredFlagSet(&Flags{})
	cmd := subcmd.NewCommand(name, fs, Runner(cmds, out), subcmd.ExactlyNumArguments(1))
	cmd.Document("generate shell completion scripts", "<shell> - shell name (bash or zsh)")
	return cmd
}

// Generate writes a shell completion script for cmds to w. If prog is empty,
// filepath.Base(os.Args[0]) is used. Supported values for shell are "bash" and
// "zsh".
func Generate(w io.Writer, shell, prog string, cmds *subcmd.CommandSet) error {
	switch shell {
	case BashShell:
		return Bash(w, prog, cmds)
	case ZshShell:
		return Zsh(w, prog, cmds)
	default:
		return fmt.Errorf("unsupported shell %q: supported shells are %s", shell, strings.Join(SupportedShells(), ", "))
	}
}

// Bash writes a bash completion script for cmds to w. If prog is empty,
// filepath.Base(os.Args[0]) is used.
func Bash(w io.Writer, prog string, cmds *subcmd.CommandSet) error {
	root, err := buildTree(prog, cmds)
	if err != nil {
		return err
	}
	return generateBash(w, root)
}

// Zsh writes a zsh completion script for cmds to w. If prog is empty,
// filepath.Base(os.Args[0]) is used.
func Zsh(w io.Writer, prog string, cmds *subcmd.CommandSet) error {
	root, err := buildTree(prog, cmds)
	if err != nil {
		return err
	}
	return generateZsh(w, root)
}

type flagSpec struct {
	name   string
	usage  string
	isBool bool
}

type node struct {
	name        string
	path        string
	description string
	flags       []flagSpec
	children    []*node
}

func buildTree(prog string, cmds *subcmd.CommandSet) (*node, error) {
	if cmds == nil {
		return nil, errors.New("command set must not be nil")
	}
	if prog == "" {
		prog = filepath.Base(os.Args[0])
	}
	if prog == "" || prog == "." {
		return nil, errors.New("program name must not be empty")
	}

	root := &node{
		name:  prog,
		path:  "",
		flags: extractFlags(cmds.GlobalFlags()),
	}

	if tl := cmds.TopLevelCommand(); tl != nil {
		root.description = tl.Description()
		root.flags = mergeFlags(root.flags, extractFlags(tl.Flags()))
		return root, nil
	}

	root.children = buildChildren("", cmds.Subcommands())

	hasHelp := hasChild(root.children, "help")
	hasVCS := hasChild(root.children, "vcsinfo")

	if !hasHelp {
		// help <cmd> completes top-level command names.
		helpChildren := make([]*node, 0, len(root.children))
		for _, ch := range root.children {
			helpChildren = append(helpChildren, &node{
				name:        ch.name,
				path:        "help/" + ch.name,
				description: ch.description,
			})
		}
		root.children = append(root.children, &node{
			name:        "help",
			path:        "help",
			description: "show help for a command",
			children:    helpChildren,
		})
	}
	if !hasVCS {
		root.children = append(root.children, &node{
			name:        "vcsinfo",
			path:        "vcsinfo",
			description: "show version control information",
		})
	}

	return root, nil
}

func buildChildren(parentPath string, cmds []*subcmd.Command) []*node {
	out := make([]*node, 0, len(cmds))
	for _, cmd := range cmds {
		p := cmd.Name()
		if parentPath != "" {
			p = parentPath + "/" + cmd.Name()
		}
		n := &node{
			name:        cmd.Name(),
			path:        p,
			description: cmd.Description(),
			flags:       extractFlags(cmd.Flags()),
		}
		if sub := cmd.Subcommands(); sub != nil {
			n.children = buildChildren(p, sub.Subcommands())
		}
		out = append(out, n)
	}
	return out
}

func hasChild(children []*node, name string) bool {
	return slices.ContainsFunc(children, func(n *node) bool {
		return n.name == name
	})
}

type boolFlag interface {
	IsBoolFlag() bool
}

func extractFlags(fs *subcmd.FlagSet) []flagSpec {
	if fs == nil || fs.FlagSet() == nil {
		return nil
	}
	var out []flagSpec
	fs.FlagSet().VisitAll(func(f *flag.Flag) {
		isBool := false
		if bf, ok := f.Value.(boolFlag); ok {
			isBool = bf.IsBoolFlag()
		}
		out = append(out, flagSpec{
			name:   f.Name,
			usage:  singleLine(f.Usage),
			isBool: isBool,
		})
	})
	sort.Slice(out, func(i, j int) bool {
		return out[i].name < out[j].name
	})
	return out
}

func mergeFlags(a, b []flagSpec) []flagSpec {
	seen := make(map[string]bool, len(a))
	out := make([]flagSpec, 0, len(a)+len(b))
	for _, f := range a {
		seen[f.name] = true
		out = append(out, f)
	}
	for _, f := range b {
		if !seen[f.name] {
			seen[f.name] = true
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].name < out[j].name
	})
	return out
}

func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func sanitizeIdent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func walkNodes(n *node, fn func(*node)) {
	fn(n)
	for _, ch := range n.children {
		walkNodes(ch, fn)
	}
}
