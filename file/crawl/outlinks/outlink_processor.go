// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks

import (
	"regexp"
	"strings"

	"cloudeng.io/text/textutil"
)

// Process is an interface for processing outlinks.
type Process interface {
	Process(outlink []string) []string
}

// RegexpProcessor is an implementation of Process that uses regular
// expressions to determine whether a link should be ignored (nofollow),
// followed or rewritten.
// Follow overrides nofollow and only links that make it through both
// nofollow and follow are rewritten. Each of the rewrites is applied
// in turn and all of the rewritten values are returned.
type RegexpProcessor struct {
	NoFollow []string // regular expressions that match links that should be ignored.
	Follow   []string // regular expressions that match links that should be followed. Follow overrides NoFollow.
	Rewrite  []string // rewrite rules that are applied to links that are followed specified as textutil.RewriteRule strings
	nofollow []*regexp.Regexp
	follow   []*regexp.Regexp
	reqwrite []textutil.RewriteRule
}

func compileRegexps(rules []string) ([]*regexp.Regexp, error) {
	result := make([]*regexp.Regexp, len(rules))
	for i, rule := range rules {
		r, err := regexp.Compile(rule)
		if err != nil {
			return nil, err
		}
		result[i] = r
	}
	return result, nil
}

// Compile is called to compile all of the regular expressions contained
// within the processor. It must be called before Process.
func (cfg *RegexpProcessor) Compile() error {
	nofollow, err := compileRegexps(cfg.NoFollow)
	if err != nil {
		return err
	}
	follow, err := compileRegexps(cfg.Follow)
	if err != nil {
		return err
	}
	rewrite := make([]textutil.RewriteRule, len(cfg.Rewrite))
	for i, rule := range cfg.Rewrite {
		r, err := textutil.NewRewriteRule(rule)
		if err != nil {
			return err
		}
		rewrite[i] = r
	}
	cfg.nofollow = nofollow
	cfg.follow = follow
	cfg.reqwrite = rewrite
	return nil
}

func matchRegexps(regexps []*regexp.Regexp, outlink string) bool {
	for _, r := range regexps {
		if r.MatchString(outlink) {
			return true
		}
	}
	return false
}

func (cfg *RegexpProcessor) Process(outlinks []string) []string {
	out := make([]string, 0, len(outlinks))
	for _, outlink := range outlinks {
		if outlink[0] == '#' {
			continue
		}
		for _, hashtag := range []string{"#", "%23"} {
			if idx := strings.LastIndex(outlink, hashtag); idx != -1 {
				outlink = outlink[:idx]
			}
		}
		nofollow := matchRegexps(cfg.nofollow, outlink)
		follow := matchRegexps(cfg.follow, outlink)
		if nofollow && !follow {
			continue
		}
		for _, r := range cfg.reqwrite {
			rewrite := r.ReplaceAllString(outlink)
			out = append(out, rewrite)
		}
	}
	return out
}

// PassthroughProcessor implements Process and simply returns its input.
type PassthroughProcessor struct{}

func (pp *PassthroughProcessor) Process(outlinks []string) []string {
	return outlinks
}
