// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import "net/url"

// URLMatcher implements Matcher for http and https paths.
func URLMatcher(p string) Match {
	url, err := url.Parse(p)
	if err != nil {
		return Match{}
	}
	if url.Scheme != "http" && url.Scheme != "https" {
		return Match{}
	}
	return Match{
		Matched:    p,
		Scheme:     url.Scheme,
		Host:       url.Host,
		Path:       url.Path,
		Key:        url.Path,
		Separator:  '/',
		Parameters: parametersFromQuery(url),
	}
}
