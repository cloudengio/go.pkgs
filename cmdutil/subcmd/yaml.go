// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"flag"
	"fmt"
	"strings"

	"cloudeng.io/cmdutil"
)

// use to separate command names at different levels.
const levelSep = "/"

func buildTree(cmdDict map[string]*Command, parent string, defs []commandDef) []*Command {
	cmds := make([]*Command, len(defs))
	for i, def := range defs {
		pathName := strings.TrimPrefix(parent+levelSep+def.Name, levelSep)
		if def.Commands == nil {
			cmd := &Command{
				name:        def.Name,
				description: def.Summary,
				arguments:   strings.Join(def.Arguments, " "),
			}
			fn := determineOptForArgs(def.Arguments)
			fn(&cmd.opts)
			cmdDict[pathName] = cmd
			cmds[i] = cmd
			continue
		}
		cmdSet := NewCommandSet(buildTree(cmdDict, parent+levelSep+def.Name, def.Commands)...)
		cmds[i] = NewCommandLevel(def.Name, cmdSet)
		cmds[i].Document(def.Summary, def.Arguments...)
		cmdDict[pathName] = cmds[i]
	}
	return cmds
}

type commandDef struct {
	Name      string
	Summary   string
	Arguments []string
	Commands  []commandDef
}

type CommandSetYAML struct {
	*CommandSet

	cmdDict map[string]*Command
}

type CurrentCommand struct {
	set *Command
	err error
}

// Set looks up the command specified by names. Each sub-command in a multi-level
// command should be specified separately. The returned CurrentCommand should
// be used to set the Runner and FlagSet to associate with that command.
func (c *CommandSetYAML) Set(names ...string) *CurrentCommand {
	cs := &CurrentCommand{}
	cs.set = c.cmdDict[strings.Join(names, "/")]
	if cs.set == nil {
		cs.err = fmt.Errorf("%v is not one of the supported commands", strings.Join(names, " "))
	}
	return cs
}

// RunnerAndFlags specifies the Runner and FlagSet for the currently 'set'
// command as returned by CommandSetYAML.Set.
func (c *CurrentCommand) RunnerAndFlags(runner Runner, fs *FlagSet) error {
	if c.err != nil {
		return c.err
	}
	c.set.runner = runner
	c.set.flags = fs
	return nil
}

// MustRunnerAndFlags is like RunnerAndFlags but will panic on error.
func (c *CurrentCommand) MustRunnerAndFlags(runner Runner, fs *FlagSet) {
	if err := c.RunnerAndFlags(runner, fs); err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}

// FromYAML parses a YAML specification of the command tree.
func FromYAML(spec []byte) (*CommandSetYAML, error) {
	var yamlCmd commandDef
	if err := cmdutil.ParseYAMLConfig(spec, &yamlCmd); err != nil {
		return nil, err
	}
	cmdSet := &CommandSetYAML{
		cmdDict: map[string]*Command{},
	}
	tlcmd := NewCommand(yamlCmd.Name, nil, nil, determineOptForArgs(yamlCmd.Arguments))
	tlcmd.Document(yamlCmd.Summary, yamlCmd.Arguments...)
	cmdSet.cmdDict[yamlCmd.Name] = tlcmd
	if yamlCmd.Commands != nil {
		cmds := buildTree(cmdSet.cmdDict, "", yamlCmd.Commands)
		cmdSet.CommandSet = NewCommandSet(cmds...)
		cmdSet.document = yamlCmd.Summary
		tlcmd.flags = &FlagSet{}
		tlcmd.flags.flagSet = flag.NewFlagSet(yamlCmd.Name, flag.ContinueOnError)
	} else {
		cmdSet.CommandSet = NewCommandSet(tlcmd)
		cmdSet.cmd = tlcmd
	}
	return cmdSet, nil
}

func determineOptForArgs(args []string) CommandOption {
	if len(args) == 0 {
		return WithoutArguments()
	}
	if args[len(args)-1] == "..." {
		return AtLeastNArguments(len(args) - 1)
	}
	if len(args) == 1 {
		a := args[0]
		if a[0] == '[' && a[len(a)-1] == ']' {
			return OptionalSingleArgument()
		}
	}
	return ExactlyNumArguments(len(args))
}

// MustFromYAML is like FromYAML but will panic if the YAML spec is
// incorrectly defined.
func MustFromYAML(spec string) *CommandSetYAML {
	cs, err := FromYAML([]byte(spec))
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	return cs
}
