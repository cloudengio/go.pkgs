// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

// ParsePermissions parses a permission string in octal ("0700", "700",
// "0o700", "0x1c0"), rwx ("rwxr-xr-x", "-rwx------", "rwx") or symbolic chmod
// ("u=rwx,go=", "u=rwx,go=rx") format.
func ParsePermissions(s string) (Permissions, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty permissions")
	}

	// 1. Explicit octal prefixes: "0o700", "0O700"
	if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		v, err := strconv.ParseUint(s[2:], 8, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid octal permissions %q: %w", s, err)
		}
		return Permissions(v), nil
	}

	// 2. Explicit hex prefixes: "0x1c0", "0X1c0"
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err := strconv.ParseUint(s[2:], 16, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid hex permissions %q: %w", s, err)
		}
		return Permissions(v), nil
	}

	// 3. Octal strings: 1 to 4 octal digits (e.g. "700", "755", "0700", "0755", "644", "0644")
	if isOctalString(s) {
		v, err := strconv.ParseUint(s, 8, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid octal permissions %q: %w", s, err)
		}
		return Permissions(v), nil
	}

	// 4. Standard 9 or 10 character rwx string: "rwxr-xr-x", "-rwx------", "drwxr-xr-x"
	if mode, ok := parseRWXString(s); ok {
		return Permissions(mode), nil
	}

	// 5. 3-character user rwx shorthand: "rwx", "r-x", "rw-"
	if mode, ok := parse3CharRWX(s); ok {
		return Permissions(mode), nil
	}

	// 6. Symbolic chmod notation: "u=rwx,go=", "u=rwx,go=rx", "a=rwx", "u+rwx"
	if mode, err := parseSymbolicPermissions(s); err == nil {
		return Permissions(mode), nil
	}

	return 0, fmt.Errorf("invalid permissions format %q: expected octal (e.g. 0700, 700), rwx (e.g. rwxr-xr-x, -rwx------), or symbolic (e.g. u=rwx,go=)", s)
}

func isOctalString(s string) bool {
	if len(s) == 0 || len(s) > 4 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '7' {
			return false
		}
	}
	return true
}

func parseTriplet(s string, shift int, specLower, specUpper rune, specMode fs.FileMode) (fs.FileMode, bool) {
	var mode fs.FileMode
	if s[0] == 'r' {
		mode |= 4 << shift
	} else if s[0] != '-' {
		return 0, false
	}
	if s[1] == 'w' {
		mode |= 2 << shift
	} else if s[1] != '-' {
		return 0, false
	}
	switch rune(s[2]) {
	case 'x':
		mode |= 1 << shift
	case specLower:
		mode |= (1 << shift) | specMode
	case specUpper:
		mode |= specMode
	case '-':
	default:
		return 0, false
	}
	return mode, true
}

func parseRWXString(s string) (fs.FileMode, bool) {
	if len(s) == 10 {
		s = s[1:]
	}
	if len(s) != 9 {
		return 0, false
	}
	u, ok1 := parseTriplet(s[0:3], 6, 's', 'S', fs.ModeSetuid)
	g, ok2 := parseTriplet(s[3:6], 3, 's', 'S', fs.ModeSetgid)
	o, ok3 := parseTriplet(s[6:9], 0, 't', 'T', fs.ModeSticky)
	if !ok1 || !ok2 || !ok3 {
		return 0, false
	}
	return u | g | o, true
}

func parse3CharRWX(s string) (fs.FileMode, bool) {
	if len(s) != 3 {
		return 0, false
	}
	var mode fs.FileMode
	if s[0] == 'r' {
		mode |= 0400
	} else if s[0] != '-' {
		return 0, false
	}
	if s[1] == 'w' {
		mode |= 0200
	} else if s[1] != '-' {
		return 0, false
	}
	if s[2] == 'x' {
		mode |= 0100
	} else if s[2] != '-' {
		return 0, false
	}
	return mode, true
}

func parseWho(whoStr, clause string) (u, g, o bool, err error) {
	if whoStr == "" || whoStr == "a" {
		return true, true, true, nil
	}
	for _, c := range whoStr {
		switch c {
		case 'u':
			u = true
		case 'g':
			g = true
		case 'o':
			o = true
		case 'a':
			u, g, o = true, true, true
		default:
			return false, false, false, fmt.Errorf("invalid who character %q in %q", c, clause)
		}
	}
	return u, g, o, nil
}

func applyWho(uBit, gBit, oBit fs.FileMode, u, g, o bool) fs.FileMode {
	var bits fs.FileMode
	if u {
		bits |= uBit
	}
	if g {
		bits |= gBit
	}
	if o {
		bits |= oBit
	}
	return bits
}

func permCharBits(c rune, u, g, o bool) (fs.FileMode, error) {
	switch c {
	case 'r':
		return applyWho(0400, 0040, 0004, u, g, o), nil
	case 'w':
		return applyWho(0200, 0020, 0002, u, g, o), nil
	case 'x':
		return applyWho(0100, 0010, 0001, u, g, o), nil
	case 's':
		return applyWho(fs.ModeSetuid, fs.ModeSetgid, 0, u, g, o), nil
	case 't':
		return applyWho(0, 0, fs.ModeSticky, u, g, o), nil
	default:
		return 0, fmt.Errorf("invalid perm character %q", c)
	}
}

func whoMask(u, g, o bool) fs.FileMode {
	var mask fs.FileMode
	if u {
		mask |= 0700 | fs.ModeSetuid
	}
	if g {
		mask |= 0070 | fs.ModeSetgid
	}
	if o {
		mask |= 0007 | fs.ModeSticky
	}
	return mask
}

func parsePermBits(permsStr string, u, g, o bool) (fs.FileMode, error) {
	var bits fs.FileMode
	for _, c := range permsStr {
		b, err := permCharBits(c, u, g, o)
		if err != nil {
			return 0, err
		}
		bits |= b
	}
	return bits, nil
}

func applySymbolicClause(mode fs.FileMode, clause string) (fs.FileMode, error) {
	clause = strings.TrimSpace(clause)
	if clause == "" {
		return 0, errors.New("empty clause in symbolic permissions")
	}
	opIdx := strings.IndexAny(clause, "=+-")
	if opIdx < 0 {
		return 0, fmt.Errorf("missing operator in clause %q", clause)
	}
	op := clause[opIdx]
	u, g, o, err := parseWho(clause[:opIdx], clause)
	if err != nil {
		return 0, err
	}
	bits, err := parsePermBits(clause[opIdx+1:], u, g, o)
	if err != nil {
		return 0, err
	}
	mask := whoMask(u, g, o)
	switch op {
	case '=':
		return (mode &^ mask) | bits, nil
	case '+':
		return mode | bits, nil
	case '-':
		return mode &^ bits, nil
	}
	return mode, nil
}

func parseSymbolicPermissions(s string) (fs.FileMode, error) {
	var mode fs.FileMode
	for _, clause := range strings.Split(s, ",") {
		var err error
		mode, err = applySymbolicClause(mode, clause)
		if err != nil {
			return 0, err
		}
	}
	return mode, nil
}
