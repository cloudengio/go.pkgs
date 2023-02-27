// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// Extension allows for extending a YAMLCommandSet with additional
// commands at runtime. Implementations of extension are used in conjunction
// with a templated version of the YAML command tree spec. The template
// can refer to an extension using the subcmdExtension function in a template
// pipeline:
//
//   - name: command
//     commands:
//     {{range subcmdExtension "exensionName"}}{{.}}
//     {{end}}
//
// The extensionName is the name of the extension as returned by the Name
// method, and . refers to results of the YAML method split into single
// lines. Thus for the above example, the YAML method can return:
//
// `- name: c3.1
// - name: c3.2`
//
// The template expansion ensures the correct indentation in the final
// YAML that's used to create the command tree.
//
// In addition to adding the extension to the YAML used to create the
// command tree, the Set method is also used to add the extension's commands
// to the command set. The Set method is called by
// CommandSetYAML.AddExtensions which should itself be called before the
// command set is used.
type Extension interface {
	Name() string
	YAML() string
	Set(cmdSet *CommandSetYAML) error
}

type extension struct {
	name     string
	spec     string
	appendFn func(cmdSet *CommandSetYAML) error
}

func (e *extension) Name() string {
	return e.name
}

func (e *extension) YAML() string {
	return e.spec
}

func (e *extension) Set(cmdSet *CommandSetYAML) error {
	return e.appendFn(cmdSet)
}

// NewExtension
func NewExtension(name, spec string, appendFn func(cmdSet *CommandSetYAML) error) Extension {
	return &extension{
		name:     name,
		spec:     spec,
		appendFn: appendFn,
	}
}

// AddExtensions calls the Set method on each of the extensions.
func (c *CommandSetYAML) AddExtensions() error {
	for _, ext := range c.extensions {
		if err := ext.Set(c); err != nil {
			return err
		}
	}
	return nil
}

// MustAddExtensions is like AddExtensions but panics on error.
func (c *CommandSetYAML) MustAddExtensions() {
	if err := c.AddExtensions(); err != nil {
		panic(fmt.Sprintf("%v", err))
	}
}

type extensionFunc struct {
	exts map[string][]string
}

func (ef *extensionFunc) extension(name string) []string {
	return ef.exts[name]
}

func yamlTemplate(specTpl string, exts ...Extension) ([]byte, error) {
	extFunc := extensionFunc{
		exts: make(map[string][]string),
	}
	for _, ext := range exts {
		extFunc.exts[ext.Name()] = strings.Split(strings.TrimSpace(ext.YAML()), "\n")
	}
	tpl, err := template.New("subcmdYaml").Funcs(template.FuncMap{
		"subcmdExtension": extFunc.extension,
	}).Parse(specTpl)
	if err != nil {
		return nil, err
	}
	extendedYAML := &bytes.Buffer{}
	if err := tpl.Execute(extendedYAML, nil); err != nil {
		return nil, err
	}
	return extendedYAML.Bytes(), nil
}

// FromYAMLTemplate returns a CommandSetYAML using the expanded value of
// the supplied template and the supplied extensions.
func FromYAMLTemplate(specTpl string, exts ...Extension) (*CommandSetYAML, []byte, error) {
	extendedYAML, err := yamlTemplate(specTpl, exts...)
	if err != nil {
		return nil, nil, err
	}
	cs, err := FromYAML(extendedYAML)
	if err != nil {
		return nil, extendedYAML, err
	}
	cs.extensions = exts
	return cs, extendedYAML, nil
}

// MustFromYAMLTemplate is like FromYAMLTemplate except that it panics
// on error.
func MustFromYAMLTemplate(specTpl string, exts ...Extension) (*CommandSetYAML, []byte) {
	cmds, expanded, err := FromYAMLTemplate(specTpl, exts...)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}
	return cmds, expanded
}

type mergedExtension struct {
	name string
	exts []Extension
	body string
}

func (me *mergedExtension) Name() string {
	return me.name
}

func (me *mergedExtension) YAML() string {
	return me.body
}

func (me *mergedExtension) Set(cmdSet *CommandSetYAML) error {
	for _, ext := range me.exts {
		if err := ext.Set(cmdSet); err != nil {
			return err
		}
	}
	return nil
}

// MergeExtensions returns an extension that merges the supplied extensions.
// Calling the Set method on the returned extension will call the Set method
// on each of the supplied extensions. The YAML method returns the concatenation
// of the YAML methods of the supplied extensions in the order that they are
// specified.
func MergeExtensions(name string, exts ...Extension) Extension {
	var body strings.Builder
	for _, ext := range exts {
		body.WriteString(ext.YAML())
	}
	return &mergedExtension{
		name: name,
		exts: exts,
		body: body.String(),
	}
}
