// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devserver

import (
	"net/url"
	"testing"
)

func TestExtractURL(t *testing.T) {
	testCases := []struct {
		line     string
		expected string
		hasErr   bool
	}{
		{"Project is running at http://localhost:8080/", "http://localhost:8080/", false},
		{"> Local: http://localhost:5173/", "http://localhost:5173/", false},
		{"> Local: https://127.0.0.1:3000", "https://127.0.0.1:3000", false},
		{"some other text http://example.com", "http://example.com", false},
		{"not a url httx\\:/\\url", "", true},
		{"malformed line ", "", true},
	}

	for _, tc := range testCases {
		u, err := extractURLAtLastSpace([]byte(tc.line))
		if (err != nil) != tc.hasErr {
			t.Errorf("extractURL(%q): got error %v, url %q, want error: %v", tc.line, err, u, tc.hasErr)
			continue
		}
		if err != nil {
			continue
		}
		if u.String() != tc.expected {
			t.Errorf("extractURL(%q): got %q, want %q", tc.line, u.String(), tc.expected)
		}
	}
}

func TestNewWebpackURLExtractor(t *testing.T) {
	extractor := NewWebpackURLExtractor(nil) // Use default regex

	testCases := []struct {
		line     string
		expected *url.URL
		hasErr   bool
	}{
		{"Local: http://localhost:8080/", &url.URL{Scheme: "http", Host: "localhost:8080", Path: "/"}, false},
		{"   Local: https://example.com/test/", &url.URL{Scheme: "https", Host: "example.com", Path: "/test/"}, false},
		{"Something else", nil, false},
		{"Not the right format", nil, false},
		{"Local: ", nil, true}, // Malformed line
	}

	for _, tc := range testCases {
		u, err := extractor([]byte(tc.line))
		if (err != nil) != tc.hasErr {
			t.Errorf("WebpackURLExtractor(%q): got error %v, want error: %v", tc.line, err, tc.hasErr)
			continue
		}
		if err != nil {
			continue
		}
		if (u == nil && tc.expected != nil) || (u != nil && tc.expected == nil) {
			t.Errorf("WebpackURLExtractor(%q): got %v, want %v", tc.line, u, tc.expected)
			continue
		}
		if u != nil && u.String() != tc.expected.String() {
			t.Errorf("WebpackURLExtractor(%q): got %q, want %q", tc.line, u.String(), tc.expected.String())
		}
	}
}

func TestNewViteURLExtractor(t *testing.T) {
	extractor := NewViteURLExtractor(nil) // Use default regex

	testCases := []struct {
		line     string
		expected *url.URL
		hasErr   bool
	}{
		{"➜  Local: http://localhost:5173/", &url.URL{Scheme: "http", Host: "localhost:5173", Path: "/"}, false},
		{"  ➜  Local:   https://127.0.0.1:3000", &url.URL{Scheme: "https", Host: "127.0.0.1:3000"}, false},
		{"Something else", nil, false},
		{"Not the right format", nil, false},
		{"➜  Local: ", nil, true}, // Malformed line
	}

	for _, tc := range testCases {
		u, err := extractor([]byte(tc.line))
		if (err != nil) != tc.hasErr {
			t.Errorf("ViteURLExtractor(%q): got error %v, want error: %v", tc.line, err, tc.hasErr)
			continue
		}
		if err != nil {
			continue
		}
		if (u == nil && tc.expected != nil) || (u != nil && tc.expected == nil) {
			t.Errorf("ViteURLExtractor(%q): got %v, want %v", tc.line, u, tc.expected)
			continue
		}
		if u != nil && u.String() != tc.expected.String() {
			t.Errorf("ViteURLExtractor(%q): got %q, want %q", tc.line, u.String(), tc.expected.String())
		}
	}
}
