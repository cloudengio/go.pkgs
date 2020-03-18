package cloudpath

import (
	"net/url"
	"strings"
)

// UnixMatcher implements Matcher for unix filenames. It returns UnixFileSystem
// for its scheme result.
func UnixMatcher(p string) *Match {
	// Handle file:// uris.
	if u, err := url.Parse(p); err == nil && u.Scheme == "file" {
		p = u.Path
	}
	if !strings.Contains(p, "/") {
		return nil
	}
	// Pretty much anything can be a unix filename, even a url.
	return &Match{
		Scheme:    UnixFileSystem,
		Separator: '/',
		Host:      "localhost",
		Path:      p,
		Local:     true,
	}
}
