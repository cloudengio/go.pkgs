// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdconfig

import "cloudeng.io/cmdutil/subcmd"

// ConfigExtensionName is the name of the extension returned by
// ConfigExtension, for use in the YAML template of a command tree:
//
//	{{range subcmdExtension "config"}}{{.}}
//	{{end}}
const ConfigExtensionName = "config"

const configExtensionYAML = `
- name: config
  summary: show the configuration, and its documentation.
  commands:
    - name: show
      summary: show the configuration, and the files it was read from.
    - name: describe
      summary: describe the configuration options.
`

const configExtensionEditYAML = `
- name: config
  summary: show, document, or edit the configuration.
  commands:
    - name: show
      summary: show the configuration, and the files it was read from.
    - name: describe
      summary: describe the configuration options.
    - name: edit
      summary: edit the configuration.
      arguments:
        - ...
`

// ConfigExtension returns an Extension that adds the 'config show',
// 'config describe', and (when cc.Editor is non-nil) 'config edit' commands,
// implemented by cc, to a command tree. It is used with ConfigHook, which makes
// the configuration available to Show.
func ConfigExtension[T any](cc ConfigCommands[T]) subcmd.Extension {
	spec := configExtensionYAML
	if cc.Editor != nil {
		spec = configExtensionEditYAML
	}
	return subcmd.NewExtension(ConfigExtensionName, spec, func(cmdSet *subcmd.CommandSetYAML) error {
		if err := cmdSet.Set("config", "show").Runner(cc.Show, &struct{}{}); err != nil {
			return err
		}
		if err := cmdSet.Set("config", "describe").Runner(cc.Describe, &struct{}{}); err != nil {
			return err
		}
		if cc.Editor == nil {
			return nil
		}
		editFlags := cc.EditFlags
		if editFlags == nil {
			editFlags = &struct{}{}
		}
		return cmdSet.Set("config", "edit").Runner(cc.Edit, editFlags)
	})
}
