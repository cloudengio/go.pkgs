// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package shells

import (
	"fmt"
	"io"
	"strings"
)

func generateZsh(w io.Writer, root *node) error {
	fn := "_" + sanitizeIdent(root.name)
	var b strings.Builder

	fmt.Fprintf(&b, "#compdef %s\n\n", root.name)
	writeZshNode(&b, fn, root)
	fmt.Fprintf(&b, "\nif (( $+functions[compdef] )); then\n    compdef %s %s\nfi\n", fn, shellSingleQuote(root.name))

	_, err := io.WriteString(w, b.String())
	return err
}

func zshFuncName(rootFn, path string) string {
	if path == "" {
		return rootFn
	}
	return rootFn + "_" + sanitizeIdent(path)
}

func writeZshNode(b *strings.Builder, rootFn string, n *node) {
	fn := zshFuncName(rootFn, n.path)
	fmt.Fprintf(b, "%s() {\n", fn)

	if len(n.children) == 0 {
		b.WriteString("    _arguments \\\n")
		writeZshFlags(b, n.flags)
		b.WriteString("        '*:filename:_files'\n")
		b.WriteString("}\n\n")
		return
	}

	b.WriteString("    local context state state_descr line\n")
	b.WriteString("    typeset -A opt_args\n\n")
	b.WriteString("    _arguments -C \\\n")
	writeZshFlags(b, n.flags)
	b.WriteString("        '1: :->cmds' \\\n")
	b.WriteString("        '*::arg:->args'\n\n")
	b.WriteString("    case \"$state\" in\n")
	b.WriteString("        cmds)\n")
	b.WriteString("            local -a commands\n")
	b.WriteString("            commands=(\n")
	for _, ch := range n.children {
		desc := zshEscapeDescribe(singleLine(ch.description))
		entry := ch.name
		if desc != "" {
			entry += ":" + desc
		}
		fmt.Fprintf(b, "                %s\n", shellSingleQuote(entry))
	}
	b.WriteString("            )\n")
	fmt.Fprintf(b, "            _describe -t commands '%s commands' commands\n", zshEscapeSingle(n.name))
	b.WriteString("            ;;\n")
	b.WriteString("        args)\n")
	b.WriteString("            case \"$line[1]\" in\n")
	for _, ch := range n.children {
		fmt.Fprintf(b, "                %s)\n", shellSingleQuote(ch.name))
		fmt.Fprintf(b, "                    %s\n", zshFuncName(rootFn, ch.path))
		b.WriteString("                    ;;\n")
	}
	b.WriteString("            esac\n")
	b.WriteString("            ;;\n")
	b.WriteString("    esac\n")
	b.WriteString("}\n\n")

	for _, ch := range n.children {
		writeZshNode(b, rootFn, ch)
	}
}

func writeZshFlags(b *strings.Builder, flags []flagSpec) {
	b.WriteString("        '(-h -help --help)'{-h,-help,--help}'[show help]' \\\n")
	for _, f := range flags {
		desc := zshEscapeArgSpec(f.usage)
		suffix := ""
		if !f.isBool {
			suffix = ":" + f.name + ":"
		}
		fmt.Fprintf(b, "        '(-%[1]s --%[1]s)'{-%[1]s,--%[1]s}'[%s]%s' \\\n", f.name, desc, suffix)
	}
}

func zshEscapeSingle(s string) string {
	return strings.ReplaceAll(s, "'", `'\''`)
}

func zshEscapeArgSpec(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		`[`, `\[`,
		`]`, `\]`,
		`:`, `\:`,
		`'`, `'\''`,
	)
	return r.Replace(s)
}

func zshEscapeDescribe(s string) string {
	return strings.ReplaceAll(s, `:`, `\:`)
}
