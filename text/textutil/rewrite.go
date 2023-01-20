// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package textutil

import (
	"fmt"
	"regexp"
	"strings"
)

// RewriteRule represents a rewrite rule of the form s/<match>/<replace>/ or
// s%<match>%<replace>%. Separators can be escpaed using a \.
type RewriteRule struct {
	Match       *regexp.Regexp
	Replacement string
}

// Match applies regexp.MatchString.
func (rr RewriteRule) MatchString(input string) bool {
	return rr.Match.MatchString(input)
}

// ReplaceAllString(input string) applies regexp.ReplaceAllString.
func (rr RewriteRule) ReplaceAllString(input string) string {
	return rr.Match.ReplaceAllString(input, rr.Replacement)
}

func toSep(input string, sep rune) (string, string, bool) {
	out := &strings.Builder{}
	var prev rune
	for i, c := range input {
		if c == sep {
			if prev != '\\' {
				return out.String(), input[i+1:], true
			}
		}
		if c != '\\' {
			out.WriteRune(c)
		}
		prev = c
	}
	return "", "", false
}

// NewReplacement accepts a string of the form s/<match-re>/<replacement>/
// or s%<match-re>%<replacement>% and returns a RewriteRule that can be used
// to perform the rewquested rewrite. Separators can be escpaed using a \.
func NewRewriteRule(rule string) (RewriteRule, error) {
	if len(rule) <= 2 {
		return RewriteRule{}, fmt.Errorf("rule must be at least 3 characters long")
	}
	if rule[0] != 's' {
		return RewriteRule{}, fmt.Errorf("rule must start with 's'")
	}
	sep := rune(rule[1])
	switch sep {
	case '%', '/':
	default:
		return RewriteRule{}, fmt.Errorf("rule must be of the form s/<match>/<replace>/ or s%%<match>%%<replace>%%")
	}

	match, rem, ok := toSep(rule[2:], sep)
	if !ok {
		return RewriteRule{}, fmt.Errorf("rule must be of the form s/<match>/<replace>/ or s%%<match>%%<replace>%%")
	}
	repl, rem, ok := toSep(rem, sep)
	if !ok || len(rem) != 0 {
		return RewriteRule{}, fmt.Errorf("rule must be of the form s/<match>/<replace>/ or s%%<match>%%<replace>%%")
	}
	re, err := regexp.Compile(match)
	if err != nil {
		return RewriteRule{}, err
	}
	return RewriteRule{Match: re, Replacement: repl}, nil
}

type RewriteRules []RewriteRule

func NewRewriteRules(rules ...string) (RewriteRules, error) {
	var rw []RewriteRule
	for _, rule := range rules {
		rr, err := NewRewriteRule(rule)
		if err != nil {
			return nil, err
		}
		rw = append(rw, rr)
	}
	return rw, nil
}

func (rw RewriteRules) ReplaceAllStringFirst(input string) string {
	for _, r := range rw {
		if r.MatchString(input) {
			return r.ReplaceAllString(input)
		}
	}
	return input
}
