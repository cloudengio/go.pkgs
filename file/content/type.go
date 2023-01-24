// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package content provides support for working with different content types.
// In particular it defines a mean of specifying content types and a registry
// for matching content types against handlerspackage content
package content

import (
	"fmt"
	"strings"
)

func head(input string, sep rune) (prefix, suffix string) {
	idx := strings.IndexRune(input, sep)
	if idx < 0 {
		return input, ""
	}
	return input[:idx], input[idx+1:]
}

// Type represents a content type specification in mime type format,
// major/minor[;parameter=value]. The major/minor part is required and the parameter
// is optional. The values used need not restricted to predefined mime types;
// ie. the values of major/minor;parameter=value are not restricted to those
// defined by the IANA.
type Type string

// ParseTypeFull parses a content type specification into its major/minor
// components and any parameter/value pairs. It returns an error if multiple
// / or ; characters are found.
func ParseTypeFull(ctype Type) (typ, par, value string, err error) {
	typ, tmp := head(string(ctype), ';')
	if strings.Count(typ, "/") != 1 {
		return "", "", "", fmt.Errorf("invalid content type: %v", ctype)
	}
	par, value = head(tmp, '=')
	if strings.ContainsRune(value, '=') {
		return "", "", "", fmt.Errorf("invalid parameter value: %v", ctype)
	}
	return
}

// ParseType is like ParseTypeFull but only returns the major/minor component.
func ParseType(ctype Type) (string, error) {
	typ, _ := head(string(ctype), ';')
	if strings.Count(typ, ";") >= 1 {
		return "", fmt.Errorf("invalid content type: %v", ctype)
	}
	return typ, nil
}
