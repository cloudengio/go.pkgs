// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devserver

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
)

var (
	webpackHostRE = regexp.MustCompile("Local:")
)

func extractURLAtLastSpace(line []byte) (*url.URL, error) {
	sp := bytes.LastIndex(line, []byte{' '})
	if sp < 0 || (sp+1 >= len(line)) {
		return nil, fmt.Errorf("malformed line: %s", line)
	}
	return url.Parse(string(line[sp+1:]))
}

// NewWebpackURLExtractor returns a URLExtractor that extracts the URL
// from lines that match the supplied regexp. If re is nil a default
// regexp that matches lines containing "Local:" is used.
// Example matching lines:
//
//	Local:     http://localhost:8080/
//
// Webpack output typically looks as follows:
//
// Compiled successfully!
//
// You can now view webapp-sample in the browser.
//
//	Local:            http://localhost:3000
//	On Your Network:  http://172.16.1.222:3000
//
// Note that the development build is not optimized.
// To create a production build, use npm run build.
//
// webpack compiled successfully
func NewWebpackURLExtractor(re *regexp.Regexp) URLExtractor {
	if re == nil {
		re = webpackHostRE
	}
	return func(line []byte) (*url.URL, error) {
		if !re.Match(line) {
			return nil, nil
		}
		return extractURLAtLastSpace(line)
	}
}
