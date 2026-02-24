// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"cloudeng.io/file"
	"gopkg.in/yaml.v3"
)

// ParseConfig will parse the yaml config in spec into the requested
// type. It provides improved error reporting via ErrorWithSource.
func ParseConfig(spec []byte, cfg any) error {
	if err := yaml.Unmarshal(spec, cfg); err != nil {
		return ErrorWithSource(spec, err)
	}
	return nil
}

// ParseConfigString is like ParseConfig but for a string.
func ParseConfigString(spec string, cfg any) error {
	return ParseConfig([]byte(spec), cfg)
}

// ParseConfigFile reads a yaml config file as per ParseConfig
// using file.FSReadFile to read the file. The use of FSReadFile allows
// for the configuration file to be read from storage system, including
// from embed.FS, instead of the local filesystem if an instance of fs.ReadFileFS
// is stored in the context.
func ParseConfigFile(ctx context.Context, filename string, cfg any) error {
	return parseConfigFile(ctx, filename, cfg, ParseConfig)
}

// ParseConfigStrict is like ParseConfig but reports an error if there
// are unknown fields in the yaml specification.
func ParseConfigStrict(spec []byte, cfg any) error {
	dec := yaml.NewDecoder(bytes.NewReader(spec))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return ErrorWithSource(spec, err)
	}
	return nil
}

// ParseConfigStringStrict is like ParseConfigString but reports an error if there
// are unknown fields in the yaml specification.
func ParseConfigStringStrict(spec string, cfg any) error {
	return ParseConfigStrict([]byte(spec), cfg)
}

// ParseConfigFileStrict is like ParseConfigFile but reports an error if there
// are unknown fields in the yaml specification.
func ParseConfigFileStrict(ctx context.Context, filename string, cfg any) error {
	return parseConfigFile(ctx, filename, cfg, ParseConfigStrict)
}

func parseConfigFile(ctx context.Context, filename string, cfg any, parser func([]byte, any) error) error {
	if len(filename) == 0 {
		return fmt.Errorf("no config file specified")
	}
	spec, err := file.FSReadFile(ctx, filename)
	if err != nil {
		return err
	}
	if err := parser(spec, cfg); err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}
	return nil
}

// ErrorWithSource returns an error that includes the yaml source
// code that was the cause of the error to help with debugging YAML
// errors.
// Note that the errors reported for the yaml parser may be inaccurate
// in terms of the lines the error is reported on. This seems to be particularly
// true for lists where errors with use of tabs to indent are often reported
// against the previous line rather than the offending one.
func ErrorWithSource(spec []byte, err error) error {
	specLines := bytes.Split(spec, []byte{'\n'})
	if yerr, ok := err.(*yaml.TypeError); ok {
		return yamlTypeErrorWithSource(specLines, yerr)
	}
	return yamlPanicErrorWithSource(specLines, err)
}

var yamlPanicErrsRE = regexp.MustCompile(`(.*)line (\d+):\s*(.*)`)

func yamlPanicErrorWithSource(specLines [][]byte, err error) error {
	sc := bufio.NewScanner(bytes.NewReader([]byte(err.Error())))
	var newError strings.Builder
	for sc.Scan() {
		errLine := sc.Text()
		matches := yamlPanicErrsRE.FindStringSubmatch(errLine)
		if len(matches) != 4 {
			newError.WriteString(errLine)
			newError.WriteRune('\n')
			continue
		}
		l, err := strconv.ParseInt(matches[2], 10, 32)
		if err != nil {
			newError.WriteString(errLine)
			newError.WriteRune('\n')
			continue
		}
		fmt.Fprintf(&newError, "%vline %d: %q: %v", matches[1], l, specLines[l-1], matches[3]) //nolint:gosec // G705: XSS via taint analysis not relevant here.
	}
	return errors.New(newError.String())
}

var yamlTypeErrsRE = regexp.MustCompile(`\s*line (\d+):\s*(.*)`)

func yamlTypeErrorWithSource(specLines [][]byte, err *yaml.TypeError) error {
	newErrors := make([]string, 0, len(err.Errors))
	for _, errLine := range err.Errors {
		matches := yamlTypeErrsRE.FindStringSubmatch(errLine)
		if len(matches) != 3 {
			newErrors = append(newErrors, errLine)
			continue
		}
		l, err := strconv.ParseInt(matches[1], 10, 32)
		if err != nil {
			newErrors = append(newErrors, errLine)
			continue
		}
		newErrors = append(newErrors, fmt.Sprintf("line %d: %q: %v", l, specLines[l-1], matches[2]))
	}
	return &yaml.TypeError{Errors: newErrors}
}
