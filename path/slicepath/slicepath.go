// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package slicepath provides utility routines for working with filenames (paths)
// across both local and distributed storage systems. As such it supports URI-like
// nameing schemes as well as local filesystem names. It represents the 'path'
// component of all of these schemes as slice of strings to simplify often
// performed operations on those paths such as finding common prefixes,
// suffixes or internal overlaps regardless of the storage system being used.
// The separator for each path is always specified as a parameter rather being
// compiled in to allow for non-local paths to be manipulated.
//
// The native path, has three components, two of which are optional:
//   - Scheme (optional): e.g. s3:// or http://
//   - Path: /a/b/c where / is a system specific separator
//   - Parameters (optional): Scheme dependent parameters that follow the path
//
// For manipulation, the path is converted to a string slice, the contents of
// which are documented with the Split function below.
package slicepath

import (
	"net/url"
	"path/filepath"
	"strings"
)

// Scheme returns the portion of the path that precedes a leading '//' or
// "" otherwise.
func Scheme(path string) string {
	if idx := strings.Index(path, "//"); idx > 0 {
		return path[:idx]
	}
	return ""
}

// Path returns the path component of path allowing for uri forms of path.
func Path(path string) string {
	if s := Scheme(path); len(s) == 0 {
		return path
	}
	uri, err := url.Parse(path)
	if err != nil {
		return path
	}
	return uri.Path
}

var emptyValues = map[string][]string{}

// Parameters returns the parameters in path, if any. If no parameters
// are present an empty (rather than nil), map is returned.
func Parameters(path string) map[string][]string {
	uri, err := url.Parse(path)
	if err != nil {
		return emptyValues
	}
	pars := uri.Query()
	r := make(map[string][]string, len(pars))
	for i, p := range pars {
		c := make([]string, len(p))
		copy(c, p)
		r[i] = c
	}
	return r
}

// Split slices path into substrings according to rules designed to retain the
// following information:
//   1. the path is absolute vs relative.
//   2. the path is a prefix or complete, where a complete path refers to a
//      single file.
//   3. a path of zero length is represented as a nil slice and not an empty slice.
//   4. any URI-like path is treated as absolute.
//
// Redundant information is discarded:
//   1. multiple consecutive instances of separator are treated as a single separator.
//
// The resulting format is as follows:
//   1. an empty path is represented by nil
//   2. a relative path, ie. one that does not start with a separator has an
//      empty string as the first item in the slice
//   3. a path that ends with a separator has an empty string as the final component
// of the path
//
// Illustrative examples:
//
//   ""         => nil                // empty
//   "/"        => ["", ""]           // absolute, incomplete
//   "./"       => [""]               // relative, incomplete
//   "/abc"     => ["", "abc"]        // absolute, complete
//   "abc"      => ["abc"]            // relative, complete
//   "/abc/"    => ["", "abc", ""]    // absolute, incomplete
//   "abc/"     => ["abc", ""]        // relative, incomplete
//   "s3://abc" => ["", "abc"]        // absolute, incomplete
func Split(path string, separator rune) []string {
	sep := false
	var slice []string
	var component string
	for _, r := range path {
		if r == separator {
			slice = append(slice, "")
		}
		break
	}
	for _, r := range path {
		if r == separator {
			if sep == false && len(component) > 0 {
				slice = append(slice, component)
				component = ""
			}
			sep = true
		} else {
			sep = false
			component += string(r)
		}
	}
	if len(component) > 0 {
		slice = append(slice, component)
	}
	if sep {
		slice = append(slice, "")
	}
	return slice
}

// Join creates a string path from the supplied components. It follows
// the rules specified for Join. It is the inverse of Split, that is,
// newPath == origPath for:
//   newPath = Join(sep, Split(origPath,sep)...)
func Join(separator rune, components ...string) string {
	sep := string(separator)
	switch len(components) {
	case 0:
		return ""
	case 1:
		if components[0] == sep {
			return sep
		}
		return components[0]
	}
	result := ""
	leading := components[0] == ""
	if leading {
		components = components[1:]
		if len(components) == 0 {
			return result
		}
		result = sep
	}

	trailing := components[len(components)-1] == ""
	if trailing {
		components = components[:len(components)-1]
	}
	if len(components) > 0 {
		result += strings.Join(components, sep)
		if trailing {
			result += sep
		}
	}
	return result
}

// Dir returns the internal components.
func Dir(path []string) []string {
	if l := len(path); l > 0 {
		return path[:l]
	}
	return nil
}

// Base returns the 'base', or 'filename' component of path, ie. the last one.
func Base(path []string) string {
	if l := len(path); l > 0 {
		return path[l-1]
	}
	return ""
}

// IsAbs returns true if the components were derived from an absolute path.
func IsAbs(components []string) bool {
	return len(components) > 0 && len(components[0]) == 0
}

// IsComplete returns if the components were derived from a complete path.
func IsComplete(components []string) bool {
	return len(components) > 0 && len(components[len(components)-1]) > 0
}

// VolumeName returns the Windows volumne name, the S3 bucket name or
// the host component of a URI, if any, found in the specified path.
func VolumeName(path string) string {
	scheme := Scheme(path)
	switch scheme {
	case "s3":
		components := Split(path, '/')
		if len(components) > 1 && components[0] == "" {
			return components[1]
		}
	default:
		if u, err := url.Parse(path); err == nil {
			return u.Host
		}
	}
	return filepath.VolumeName(Path(path))
}

// fio
// How to make glob work.
//func Glob(ctx *context.Context, ...)

/*
func HasSuffix(path, suffix []string) bool {

}

func TrimSuffix(path, suffix []string) []string {

}

func HasPrefix(path, prefix []string) bool {

}

func TrimPrefix(path, prefix []string) []string {

}*/
