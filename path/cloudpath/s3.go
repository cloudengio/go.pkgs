// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"net/url"
	"strings"
)

// pathFromHostAndPath will construct a / separated string from a h
// and path that starts with a /.
func pathFromHostAndPath(u *url.URL) string {
	path := ""
	if len(u.Host) > 0 {
		path += "/" + u.Host
	}
	if len(u.Path) > 0 {
		path += u.Path
	}
	return path
}

// AWSS3Matcher implements Matcher for AWS S3 object names. It returns AWSS3
// for its scheme result.
func AWSS3Matcher(p string) *Match {
	u, err := url.Parse(p)
	if err != nil {
		return nil
	}
	m := &Match{
		Scheme:     AWSS3,
		Separator:  '/',
		Parameters: parametersFromQuery(u),
	}
	switch u.Scheme {
	case "s3":
		m.Volume = u.Host
		m.Path = pathFromHostAndPath(u)
		return m
	case "http", "https":
		m.Host = u.Host
		m.Path = u.Path
	default:
		return nil
	}
	leading := strings.TrimSuffix(u.Host, ".amazonaws.com")
	if len(leading) == len(u.Host) {
		// not trimmed.
		return nil
	}
	parts := strings.Split(leading, ".")
	if len(parts) == 2 && parts[0] == "s3" {
		// https://s3.Region.amazonaws.com/bucket-name/key
		m.Volume = firstPathComponent(u.Path)
		return m
	}
	if len(parts) > 2 && parts[len(parts)-2] == "s3" {
		// https://bucket.name.s3.Region.amazonaws.com/key
		m.Volume = leading[:strings.Index(leading, "s3")-1]
		return m
	}
	return nil
}
