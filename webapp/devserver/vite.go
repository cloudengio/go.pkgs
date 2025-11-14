// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devserver

import (
	"net/url"
	"regexp"
)

var (
	viteHostRE = regexp.MustCompile("➜  Local:")
)

// NewViteURLExtractor returns a URLExtractor that extracts the URL
// from lines that match the supplied regexp. If re is nil a default
// regexp that matches lines starting with   "➜  Local:" is used.
// Example matching lines:
//
//	➜  Local:   http://localhost:5173/
//
// Vite output typically looks as follows:
//
//	> webapp-sample-vite@0.0.0 dev
//	> vite --host
//
//
//	  ROLLDOWN-VITE v7.1.14  ready in 71 ms
//
//	  ➜  Local:   http://localhost:5173/
//	  ➜  Network: http://172.16.1.222:5173/
//	  ➜  Network: http://172.16.1.142:5173/
func NewViteURLExtractor(re *regexp.Regexp) URLExtractor {
	if re == nil {
		re = viteHostRE
	}
	return func(line []byte) (*url.URL, error) {
		if !re.Match(line) {
			return nil, nil
		}
		return extractURLAtLastSpace(line)
	}
}
