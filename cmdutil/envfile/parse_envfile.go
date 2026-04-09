// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package envfile parses files containing shell-style environment variable
// definitions of the form NAME=VALUE, as commonly used in .env files.
package envfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// Parse reads environment variable definitions from r and returns a map of
// variable names to their values. Each non-blank, non-comment line must be
// of the form:
//
//	[export] NAME=VALUE
//
// VALUE may be:
//   - Unquoted: terminated by an unquoted # (inline comment) or end of line;
//     leading and trailing whitespace is trimmed.
//   - Single-quoted ('VALUE'): literal content, no escape processing.
//   - Double-quoted ("VALUE"): backslash escapes \n \t \r \" \\ \$ are
//     interpreted; other \X sequences are passed through as-is.
//
// Lines whose first non-whitespace character is # are comments and are
// skipped. The export keyword prefix is accepted and ignored.
// Lines without an = are silently skipped.
// If a name appears more than once the last value wins.
func Parse(r io.Reader) (map[string]string, error) {
	result := map[string]string{}
	br := bufio.NewReader(r)
	for lineNum := 1; ; lineNum++ {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			name, value, ok, perr := parseLine(line)
			if perr != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, perr)
			}
			if ok {
				result[name] = value
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// ParseFile is a convenience wrapper around Parse that opens and reads a file.
func ParseFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func parseLine(line string) (name, value string, ok bool, err error) {
	// Strip leading whitespace.
	line = strings.TrimLeftFunc(line, unicode.IsSpace)

	// Skip blank lines and comment lines.
	if len(line) == 0 || line[0] == '#' {
		return "", "", false, nil
	}

	// Strip optional 'export' keyword.
	if rest, hasExport := stripExport(line); hasExport {
		line = rest
	}

	// Split on the first '='.
	var rest string
	name, rest, ok = strings.Cut(line, "=")
	if !ok {
		// No '=': treat as a bare export or unknown directive; skip.
		return "", "", false, nil
	}

	if !isValidName(name) {
		return "", "", false, fmt.Errorf("invalid variable name %q", name)
	}

	value, err = parseValue(rest)
	if err != nil {
		return "", "", false, err
	}
	return name, value, true, nil
}

// stripExport removes a leading "export" keyword followed by whitespace.
// Returns the remainder and true if the keyword was present.
func stripExport(s string) (string, bool) {
	if !strings.HasPrefix(s, "export") {
		return s, false
	}
	rest := s[len("export"):]
	if len(rest) == 0 || !unicode.IsSpace(rune(rest[0])) {
		return s, false
	}
	return strings.TrimLeftFunc(rest, unicode.IsSpace), true
}

// isValidName reports whether s is a valid shell variable name:
// [A-Za-z_][A-Za-z0-9_]*
func isValidName(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
		} else {
			if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return false
			}
		}
	}
	return true
}

// parseValue dispatches to the appropriate parser based on the opening
// character of the value.
func parseValue(s string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}
	switch s[0] {
	case '\'':
		return parseSingleQuoted(s[1:])
	case '"':
		return parseDoubleQuoted(s[1:])
	default:
		return parseUnquoted(s), nil
	}
}

// parseSingleQuoted reads until the closing ' with no escape processing.
func parseSingleQuoted(s string) (string, error) {
	value, _, ok := strings.Cut(s, "'")
	if !ok {
		return "", fmt.Errorf("unterminated single-quoted value")
	}
	return value, nil
}

// parseDoubleQuoted reads until an unescaped " and processes backslash
// escapes: \n \t \r \" \\ \$ are interpreted; other \X are kept as-is.
func parseDoubleQuoted(s string) (string, error) {
	var b strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '"':
			return b.String(), nil
		case c == '\\' && i+1 < len(s):
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case '$':
				b.WriteByte('$')
			default:
				// Unrecognised escape: keep the backslash.
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}
		default:
			b.WriteByte(c)
		}
		i++
	}
	return "", fmt.Errorf("unterminated double-quoted value")
}

// parseUnquoted reads an unquoted value. A # character starts an inline
// comment only when preceded by whitespace, matching bash behaviour:
//
//	FOO=value # comment  →  "value"
//	FOO=#hash            →  "#hash"
//	FOO=a#b              →  "a#b"
//
// Leading and trailing whitespace are trimmed from the result.
func parseUnquoted(s string) string {
	end := len(s)
	for i, c := range s {
		if c == '#' && i > 0 && unicode.IsSpace(rune(s[i-1])) {
			end = i
			break
		}
	}
	return strings.TrimFunc(s[:end], unicode.IsSpace)
}
