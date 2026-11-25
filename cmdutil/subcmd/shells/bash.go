// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package shells

import (
	"fmt"
	"io"
	"strings"
)

func generateBash(w io.Writer, root *node) error {
	fn := "_" + sanitizeIdent(root.name) + "_completions"
	var b strings.Builder

	fmt.Fprintf(&b, "# bash completion for %s\n", root.name)
	fmt.Fprintf(&b, "%s() {\n", fn)
	b.WriteString("    local cur prev word path expect_val i\n")
	b.WriteString("    COMPREPLY=()\n")
	b.WriteString("    cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	b.WriteString("    path=\"\"\n")
	b.WriteString("    expect_val=0\n\n")
	b.WriteString("    for ((i = 1; i < COMP_CWORD; i++)); do\n")
	b.WriteString("        word=\"${COMP_WORDS[i]}\"\n")
	b.WriteString("        if [[ $expect_val -eq 1 ]]; then\n")
	b.WriteString("            expect_val=0\n")
	b.WriteString("            continue\n")
	b.WriteString("        fi\n")
	b.WriteString("        if [[ \"$word\" == -* ]]; then\n")
	b.WriteString("            if [[ \"$word\" != *=* ]]; then\n")
	b.WriteString("                local flag_name=\"${word#--}\"\n")
	b.WriteString("                flag_name=\"${flag_name#-}\"\n")
	b.WriteString("                case \"$path:$flag_name\" in\n")

	var valueFlagCases []string
	walkNodes(root, func(n *node) {
		for _, f := range n.flags {
			if !f.isBool {
				valueFlagCases = append(valueFlagCases, shellSingleQuote(n.path+":"+f.name))
			}
		}
	})
	if len(valueFlagCases) > 0 {
		fmt.Fprintf(&b, "                    %s)\n", strings.Join(valueFlagCases, "|"))
		b.WriteString("                        expect_val=1\n")
		b.WriteString("                        ;;\n")
	}
	b.WriteString("                esac\n")
	b.WriteString("            fi\n")
	b.WriteString("            continue\n")
	b.WriteString("        fi\n")
	b.WriteString("        case \"$path:$word\" in\n")
	walkNodes(root, func(n *node) {
		for _, ch := range n.children {
			fmt.Fprintf(&b, "            %s:%s)\n", bashQuoteCase(n.path), bashQuoteCase(ch.name))
			fmt.Fprintf(&b, "                path=%s\n", shellSingleQuote(ch.path))
			b.WriteString("                ;;\n")
		}
	})
	b.WriteString("            *)\n")
	b.WriteString("                ;;\n")
	b.WriteString("        esac\n")
	b.WriteString("    done\n\n")

	b.WriteString("    if [[ $expect_val -eq 1 ]]; then\n")
	b.WriteString("        return 0\n")
	b.WriteString("    fi\n\n")

	b.WriteString("    if [[ \"$cur\" == -* ]]; then\n")
	b.WriteString("        local flags=\"\"\n")
	b.WriteString("        case \"$path\" in\n")
	walkNodes(root, func(n *node) {
		opts := bashFlagWords(n.flags)
		fmt.Fprintf(&b, "            %s)\n", bashQuoteCase(n.path))
		fmt.Fprintf(&b, "                flags=%s\n", shellSingleQuote(opts))
		b.WriteString("                ;;\n")
	})
	b.WriteString("        esac\n")
	b.WriteString("        COMPREPLY=($(compgen -W \"$flags\" -- \"$cur\"))\n")
	b.WriteString("        return 0\n")
	b.WriteString("    fi\n\n")

	b.WriteString("    local cmds=\"\"\n")
	b.WriteString("    case \"$path\" in\n")
	walkNodes(root, func(n *node) {
		if len(n.children) == 0 {
			return
		}
		names := make([]string, len(n.children))
		for i, ch := range n.children {
			names[i] = ch.name
		}
		fmt.Fprintf(&b, "        %s)\n", bashQuoteCase(n.path))
		fmt.Fprintf(&b, "            cmds=%s\n", shellSingleQuote(strings.Join(names, " ")))
		b.WriteString("            ;;\n")
	})
	b.WriteString("    esac\n")
	b.WriteString("    if [[ -n \"$cmds\" ]]; then\n")
	b.WriteString("        COMPREPLY=($(compgen -W \"$cmds\" -- \"$cur\"))\n")
	b.WriteString("    fi\n")
	b.WriteString("}\n")
	fmt.Fprintf(&b, "complete -o default -F %s %s\n", fn, shellSingleQuote(root.name))

	_, err := io.WriteString(w, b.String())
	return err
}

func bashFlagWords(flags []flagSpec) string {
	words := make([]string, 0, len(flags)*2+3)
	for _, f := range flags {
		words = append(words, "-"+f.name, "--"+f.name)
	}
	words = append(words, "-h", "-help", "--help")
	return strings.Join(words, " ")
}

func bashQuoteCase(s string) string {
	if s == "" {
		return `""`
	}
	return shellSingleQuote(s)
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
