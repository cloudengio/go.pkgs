// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"net/url"
	"strings"
)

// return 'Region' from s3.Region.amazonaws.com
func s3Region(host string) string {
	if len(host) < 3 || host[0:3] != "s3." {
		return ""
	}
	eidx := strings.Index(host, ".amazonaws.com")
	if eidx < 0 {
		return ""
	}
	return host[3:eidx]
}

// AWSS3Matcher implements Matcher for AWS S3 object names
// assuming '/' as the separator.
// It returns AWSS3 for its scheme result.
func AWSS3Matcher(p string) Match {
	return AWSS3MatcherSep(p, '/')
}

func AWSS3MatcherSep(p string, sep byte) Match {
	m := Match{
		Matched:   p,
		Scheme:    AWSS3,
		Separator: rune(sep),
	}
	if len(p) >= 5 && p[0:5] == "s3://" {
		m.Path = p[5:]
		m.Volume, m.Key = bucketAndKey(m.Path, byte(m.Separator))
		return m
	}
	u, err := url.Parse(p)
	if err != nil {
		return Match{}
	}
	m.Parameters = parametersFromQuery(u)
	switch u.Scheme {
	case "http", "https":
	default:
		return Match{}
	}
	m.Host = u.Host
	m.Path = u.Path
	s3idx := strings.Index(u.Host, "s3.")
	if s3idx < 0 {
		return Match{}
	}
	m.Region = s3Region(u.Host[s3idx:])
	if s3idx == 0 {
		// https://s3.Region.amazonaws.com/bucket-name/key
		m.Volume, m.Key = bucketAndKey(u.Path, byte(m.Separator))
		return m
	}
	// https://bucket.name.s3.Region.amazonaws.com/key
	m.Volume = u.Host[:s3idx-1]
	m.Path = u.Path
	m.Key = u.Path
	if len(m.Key) > 0 && m.Key[0] == byte(m.Separator) {
		m.Key = m.Key[1:]
	}
	return m
}
