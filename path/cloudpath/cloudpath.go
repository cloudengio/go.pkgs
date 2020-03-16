// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package cloudpath provides utility routines for working with paths
// across both local and distributed storage systems. The set of schemes
// supported can be extended by providing additional implementations of
// the Matcher function. A cloudpath encodes two types of information:
//   1. the path name itself which can be used to access the data it names.
//   2. metadata about the where that filename is hosted.
//
// For example, s3://my-bucket/a/b, contains the path '/my-bucket/a/b' as
// well the indication that this path is hosted on S3. Most cloud storage
// systems either use URI formats natively or support their use. Both AWS S3
// and Google Cloud Storage support URLs: eg. storage.cloud.google.com/bucket/obj.
//
// cloudpath provides operations for extracting both metadata and the path
// component, and operations for working with the extracted path directly.
// A common usage is to determine the 'scheme' (eg. s3, windows, unix etc) of
// a filename and to then operate on it appropriately.
// cloudpath represents a 'path' as a slice of strings to simplify often
// performed operations such as finding common prefixes, suffixes that are
// aware of the structure of the path. For example it should be possible to easily
// determine that s3://bucket/a/b is a prefix of https://s3.us-west-2.amazonaws.com/bucket/a/b/c.
//
// All of the metadata for a path is represented using the Match type.
//
// For manipulation, the path is converted to a string slice, the contents of
// which are documented by the Split function below.
package cloudpath

import (
	"strings"
	"unicode/utf8"
)

// Split slices path into substrings according to rules designed to retain the
// following information:
//   1. the path is absolute vs relative.
//   2. the path is a prefix or a filepath.
//   3. a path of zero length is represented as a nil slice and not an empty slice.
//
// Redundant information is discarded:
//   1. multiple consecutive instances of separator are treated as a single separator.
//
// The resulting format is as follows:
//   1. an empty path is represented by nil
//   2. a relative path, ie. one that does not start with a separator has an
//      empty string as the first item in the slice
//   3. a path that ends with a separator has an empty string as the final component
//      of the path
//
// For example:
//
//   ""         => nil                // empty
//   "/"        => ["", ""]           // absolute, prefix
//   "./"       => [""]               // relative, prefix
//   "/abc"     => ["", "abc"]        // absolute, filepath
//   "abc"      => ["abc"]            // relative, filepath
//   "/abc/"    => ["", "abc", ""]    // absolute, prefix
//   "abc/"     => ["abc", ""]        // relative, prefix
func Split(path string, separator rune) []string {
	sep := false
	var slice []string
	var component string
	if r, _ := utf8.DecodeRuneInString(path); r == separator {
		slice = append(slice, "")
	}
	for _, r := range path {
		if r == separator {
			if !sep && len(component) > 0 {
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

// SplitPath calls Split with the results of cloudpath.Split(path).
func SplitPath(path string) []string {
	p, s := Path(path)
	return Split(p, s)
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
		return components[0]
	}
	result := ""
	leading := components[0] == ""
	if leading {
		components = components[1:]
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

// Prefix returns prefix components of a path.
func Prefix(path []string) []string {
	l := len(path)
	if l == 0 || len(path[l-1]) == 0 {
		return path
	}
	// remove trailing filename and mark the path as a prefix.
	if p := path[:l-1]; len(p) > 0 {
		return append(p, "")
	}
	return nil
}

// Base returns the 'base', or 'filename' component of path, ie. the last one.
func Base(path []string) string {
	l := len(path)
	if l == 0 || len(path[l-1]) == 0 {
		return ""
	}
	return path[l-1]
}

// AsPrefix returns path as a path prefix if it is not already one.
func AsPrefix(path ...string) []string {
	if l := len(path); l == 0 || len(path[l-1]) == 0 {
		return path
	}
	return append(path, "")
}

// IsAbsolute returns true if the components were derived from an absolute path.
func IsAbsolute(components []string) bool {
	return len(components) > 0 && len(components[0]) == 0
}

// IsFilepath returns true if the components were derived from a filepath.
func IsFilepath(components []string) bool {
	return len(components) > 0 && len(components[len(components)-1]) > 0
}

// Trim a trailing "" indicating that the path is a prefix so that it will
// match an internal separator. That is so, that, /a/ can be a prefix of
// /a/b/ even though they are represented as {"", "b", ""} and {"", "a", "b", ""},
// respectively.
func trimPrefixPath(path []string) (bool, []string) {
	if len(path) > 1 && len(path[len(path)-1]) == 0 {
		return true, path[:len(path)-1]
	}
	return false, path
}

// hasPrefix returns true if path has the specified prefix.
func hasPrefix(path, prefix []string) bool {
	switch {
	case len(path) == 0 && len(prefix) == 0:
		return true
	case len(prefix) == 0:
		return true
	case len(path) == 0:
		return false
	}
	for i, c := range prefix {
		if c != path[i] {
			return false
		}
	}
	return true
}

// HasPrefix returns true if path has the specified prefix.
func HasPrefix(path, prefix []string) bool {
	isPrefixPath, trimmed := trimPrefixPath(prefix)
	if isPrefixPath && len(path) == len(trimmed) {
		return false
	}
	return hasPrefix(path, trimmed)
}

// TrimPrefix removes the specified prefix from path. It returns nil
// if path and suffix are identical.
func TrimPrefix(path, prefix []string) []string {
	if len(prefix) == 0 {
		return path
	}
	isPrefixPath, trimmed := trimPrefixPath(prefix)
	if (isPrefixPath && len(path) == len(trimmed)) || !hasPrefix(path, trimmed) {
		return path
	}
	if isPrefixPath {
		return path[len(trimmed):]
	}
	if p := path[len(prefix):]; len(p) > 0 {
		return append([]string{""}, p...)
	}
	return nil
}

// HasSuffix returns true if path has the specified suffix.
func HasSuffix(path, suffix []string) bool {
	switch {
	case len(path) == 0 && len(suffix) == 0:
		return true
	case len(suffix) == 0:
		return true
	case len(path) == 0:
		return false
	}
	j := len(path) - 1
	for i := len(suffix) - 1; i >= 0; i-- {
		if suffix[i] != path[j] {
			return false
		}
		j--
	}
	return true
}

// TrimSuffix removes the specified suffix from path. It returns nil
// if path and suffix are identical.
func TrimSuffix(path, suffix []string) []string {
	if !HasSuffix(path, suffix) {
		return path
	}
	if p := path[:len(path)-len(suffix)]; len(p) > 0 {
		if len(p) == 1 && len(p[0]) == 0 && len(path[0]) == 0 {
			// special case of being left with just the separator, e.g. '/'
			return []string{"", ""}
		}
		return p
	}
	return nil
}

// returns true if the components at the specified option matched and if whether
// a mismatch is because components were not the same or because one or more path
// ended.
func sameAtLeadingOffset(paths [][]string, offset int) (matched, remaining bool) {
	val := paths[0][offset]
	for _, path := range paths[1:] {
		if len(path) <= offset {
			return false, false
		}
		if val != path[offset] {
			return false, true
		}
	}
	return true, false
}

func LongestCommonPrefix(paths ...[]string) []string {
	switch len(paths) {
	case 0:
		return []string{}
	case 1:
		return paths[0]
	}
	if len(paths[0]) == 0 {
		return []string{}
	}
	prefix := []string{}
	offset := 0
	for {
		matched, remaining := sameAtLeadingOffset(paths, offset)
		if !matched {
			if remaining && len(prefix) > 0 {
				// if the prefix is a partial match, then mark it
				// as being a prefix rather than a full match.
				prefix = append(prefix, "")
			}
			break
		}
		prefix = append(prefix, paths[0][offset])
		offset++
	}
	return prefix
}

func sameAtReverseOffset(paths [][]string, offset int) bool {
	first := paths[0]
	if len(first) < offset {
		return false
	}
	val := paths[0][len(first)-offset]
	for _, path := range paths[1:] {
		if len(path) < offset {
			return false
		}
		if val != path[len(path)-offset] {
			return false
		}
	}
	return true
}

func LongestCommonSuffix(paths ...[]string) []string {
	switch len(paths) {
	case 0:
		return []string{}
	case 1:
		return paths[0]
	}
	suffix := []string{}
	val := paths[0]
	for offset := 1; sameAtReverseOffset(paths, offset); offset++ {
		suffix = append([]string{val[len(val)-offset]}, suffix...)
	}
	if len(suffix) == 1 && len(suffix[0]) == 0 {
		return []string{}
	}
	return suffix
}
