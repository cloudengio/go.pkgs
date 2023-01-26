// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"net/url"
	"unicode/utf8"
)

// DefaultMatchers represents the built in set of Matchers.
var DefaultMatchers MatcherSpec = []Matcher{
	AWSS3Matcher,
	GoogleCloudStorageMatcher,
	URLMatcher,
	WindowsMatcher,
	UnixMatcher,
}

// MatcherSpec represents a set of Matchers that will be applied in order.
// The ordering is important, the most specific matchers need to be applied
// first. For example a matcher for Windows should precede that for a Unix
// filesystem since the latter can accept filenames in Windows format.
type MatcherSpec []Matcher

// Match is the result of a successful match.
type Match struct {
	// Original is the original string that was matched.
	Matched string
	// Scheme uniquely identifies the filesystem being used, eg. s3 or windows.
	Scheme string
	// Local is true for local filesystems.
	Local bool
	// Host will be 'localhost' for local filesystems, the host encoded
	// in a URL or otherwise empty if there is no notion of a host.
	Host string
	// Volume will be the bucket or file system share for systems that support
	// that concept, or an empty string otherwise.
	Volume string
	// Path is the filesystem path or filename to the data. It may be a prefix
	// on a cloud based system or a directory on a local one.
	Path string
	// Key is like Path except without the volume for systems where the volume
	// can appear in the path name.
	Key string
	// Region is the region for cloud based systems.
	Region string
	// Separator is the filesystem separator (e.g / or \ for windows).
	Separator rune
	// Parameters are any parameters encoded in a URL/URI based name.
	Parameters map[string][]string
}

// Matcher is the prototype for functions that parse the supplied path to determine
// if it matches a specific scheme and then breaks out the metadata encoded in the
// path. If Match.Matched is empty then no match has been found.
// Matchers for local filesystems should return "" for the host.
type Matcher func(p string) Match

const (
	// AWSS3 is the scheme for Amazon Web Service's S3 object store.
	AWSS3 = "s3"
	// GoogleCloudStorage is the scheme for Google's Cloud Storage object store.
	GoogleCloudStorage = "GoogleCloudStorage"
	// UnixFileSystem is the scheme for unix like systems such as linux, macos etc.
	UnixFileSystem = "unix"
	// WindowsFileSystem is the scheme for msdos and windows filesystems.
	WindowsFileSystem = "windows"
	// HTTP is the scheme for http.
	HTTP = "http"
	// HTTPS is the scheme for https.
	HTTPS = "https"
)

// Scheme calls DefaultMatchers.Scheme(path).
func Scheme(path string) string {
	return DefaultMatchers.Scheme(path)
}

// Volume calls DefaultMatchers.Volume(path).
func Volume(path string) string {
	return DefaultMatchers.Volume(path)
}

// Host calls DefaultMatchers.Host(path).
func Host(path string) string {
	return DefaultMatchers.Host(path)
}

// Path calls DefaultMatchers.Path(path).
func Path(path string) (string, rune) {
	return DefaultMatchers.Path(path)
}

// Key calls DefaultMatchers.Key(path).
func Key(path string) (string, rune) {
	return DefaultMatchers.Key(path)
}

// Region calls DefaultMatchers.Region(path).
func Region(url string) string {
	return DefaultMatchers.Region(url)
}

// Parameters calls DefaultMatchers.Parameters(path).
func Parameters(path string) map[string][]string {
	return DefaultMatchers.Parameters(path)
}

// IsLocal calls DefaultMatchers.IsLocal(path).
func IsLocal(path string) bool {
	return DefaultMatchers.IsLocal(path)
}

// Match applies all of the matchers in turn to match the supplied path.
func (ms MatcherSpec) Match(p string) Match {
	for _, fn := range ms {
		if m := fn(p); len(m.Matched) > 0 {
			return m
		}
	}
	return Match{}
}

// Scheme returns the portion of the path that precedes a leading '//' or
// "" otherwise.
func (ms MatcherSpec) Scheme(path string) string {
	if m := ms.Match(path); len(m.Matched) > 0 {
		return m.Scheme
	}
	return ""
}

// Host returns the host component of the path if there is one.
func (ms MatcherSpec) Host(path string) string {
	if m := ms.Match(path); len(m.Matched) > 0 {
		return m.Host
	}
	return ""
}

// Volume returns the filesystem specific volume, if any, encoded
// in the path.
func (ms MatcherSpec) Volume(path string) string {
	if m := ms.Match(path); len(m.Matched) > 0 {
		return m.Volume
	}
	return ""
}

// Path returns the path component of path and the separator to use for it.
func (ms MatcherSpec) Path(path string) (string, rune) {
	if m := ms.Match(path); len(m.Matched) > 0 {
		return m.Path, m.Separator
	}
	return "", utf8.RuneError
}

// Key returns the key component of path and the separator to use for it.
func (ms MatcherSpec) Key(path string) (string, rune) {
	if m := ms.Match(path); len(m.Matched) > 0 {
		return m.Key, m.Separator
	}
	return "", utf8.RuneError
}

// Region returns the region component for cloud based systems.
func (ms MatcherSpec) Region(url string) string {
	if m := ms.Match(url); len(m.Matched) > 0 {
		return m.Region
	}
	return ""
}

var emptyValues = map[string][]string{}

// Parameters returns the parameters in path, if any. If no parameters
// are present an empty (rather than nil), map is returned.
func (ms *MatcherSpec) Parameters(path string) map[string][]string {
	if m := ms.Match(path); len(m.Matched) > 0 && m.Parameters != nil {
		return m.Parameters
	}
	return emptyValues
}

// IsLocal returns true if the path is for a local filesystem.
func (ms *MatcherSpec) IsLocal(path string) bool {
	if m := ms.Match(path); len(m.Matched) > 0 {
		return m.Local
	}
	return false
}

// return the
func parametersFromQuery(u *url.URL) map[string][]string {
	pars := u.Query()
	if len(pars) == 0 {
		return nil
	}
	r := make(map[string][]string, len(pars))
	for i, p := range pars {
		c := make([]string, len(p))
		copy(c, p)
		r[i] = c
	}
	return r
}
