// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import "regexp"

// ConfigManager provides a means to manage configurations based on regex
// patterns that can be matched against test names. It is useful
// for centralizing the configuration of tests, especially those that
// are externalized by a one package for use by multiple others. For
// example when an interface has multiple implementations for which
// tests can be shared.
type ConfigManager[T any] struct {
	cfg        []T
	res        []*regexp.Regexp
	defaultCfg T
}

// SetDefault sets the default configuration to be returned when no regex matches.
func (c *ConfigManager[T]) SetDefault(config T) {
	c.defaultCfg = config
}

// Set associates a regex pattern with a specific configuration.
func (c *ConfigManager[T]) Set(re *regexp.Regexp, config T) {
	c.cfg = append(c.cfg, config)
	c.res = append(c.res, re)
}

// Get returns the configuration associated with the first regex that matches
// the input string. The regexes are evaluated in the order they were added via Set.
// If no regex matches, the default configuration is returned, hence there is
// no need to use a regex that matches all strings as the default case.
func (c *ConfigManager[T]) Get(s string) T {
	for i, re := range c.res {
		if re.MatchString(s) {
			return c.cfg[i]
		}
	}
	return c.defaultCfg
}
