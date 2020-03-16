package cloudpath

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

// DefaultMatchers represents the built in set of Matchers.
var DefaultMatchers MatcherSpec = []Matcher{
	AWSS3Matcher,
	GoogleCloudStorageMatcher,
	WindowsMatcher,
	UnixMatcher,
}

// MatcherSpec represents a set of Matchers and local file system schemes.
type MatcherSpec []Matcher

// Match is the result of a successful match.
type Match struct {
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
	// Separator is the filesystem separator (e.g / or \ for windows).
	Separator rune
	// Parameters are any parameters encoded in a URL/URI based name.
	Parameters map[string][]string
}

// Matcher is the prototype for functions that parse the supplied path to determine
// if it matches a specific scheme and then breaks out the metadata encoded in the
// path. Matchers for local filesystems should return "localhost" for the host.
type Matcher func(p string) *Match

const (
	// AWSS3 is the scheme for Amazon Web Service's S3 object store.
	AWSS3 = "s3"
	// GoogleCloudStorage is the scheme for Google's Cloud Storage object store.
	GoogleCloudStorage = "GoogleCloudStorage"
	// LocalUnixFileSystem is the scheme for unix like systems such as linux, macos etc.
	UnixFileSystem = "unix"
	// WindowsFileSystem is the scheme for msdos and windows filesystems.
	WindowsFileSystem = "windows"
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

// Parameters calls DefaultMatchers.Parameters(path).
func Parameters(path string) map[string][]string {
	return DefaultMatchers.Parameters(path)
}

// IsLocal calls DefaultMatchers.IsLocal(path).
func IsLocal(path string) bool {
	return DefaultMatchers.IsLocal(path)
}

// Match applies all of the matchers in turn to match the supplied path.
func (ms MatcherSpec) Match(p string) *Match {
	for _, fn := range ms {
		if m := fn(p); m != nil {
			return m
		}
	}
	return nil
}

// Scheme returns the portion of the path that precedes a leading '//' or
// "" otherwise.
func (ms MatcherSpec) Scheme(path string) string {
	if m := ms.Match(path); m != nil {
		return m.Scheme
	}
	return ""
}

// Hoost returns the host component of the path if there is one.
func (ms MatcherSpec) Host(path string) string {
	if m := ms.Match(path); m != nil {
		return m.Host
	}
	return ""
}

// Volume returns the filesystem specific volume, if any, encoded
// in the path.
func (ms MatcherSpec) Volume(path string) string {
	if m := ms.Match(path); m != nil {
		return m.Volume
	}
	return ""
}

// Path returns the path component of path and the separator to use for it.
func (ms MatcherSpec) Path(path string) (string, rune) {
	if m := ms.Match(path); m != nil {
		return m.Path, m.Separator
	}
	return "", utf8.RuneError
}

var emptyValues = map[string][]string{}

// Parameters returns the parameters in path, if any. If no parameters
// are present an empty (rather than nil), map is returned.
func (ms *MatcherSpec) Parameters(path string) map[string][]string {
	if m := ms.Match(path); m != nil && m.Parameters != nil {
		return m.Parameters
	}
	return emptyValues
}

// IsLocal returns true if the path is for a local filesystem.
func (ms *MatcherSpec) IsLocal(path string) bool {
	if m := ms.Match(path); m != nil {
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

// return the first non-empty path component in a / separate path.
func firstPathComponent(path string) string {
	for _, p := range strings.Split(path, "/") {
		if len(p) > 0 {
			return p
		}
	}
	return ""
}
