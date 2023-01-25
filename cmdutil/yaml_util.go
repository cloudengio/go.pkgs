// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseYAMLConfig will parse the yaml config in spec into the requested
// type. It provides improved error reporting by mixing in the actual
// yaml input (from the spec) into the error messages.
func ParseYAMLConfig[T any](spec []byte, cfg T) error {
	if err := yaml.Unmarshal(spec, cfg); err != nil {
		return YAMLErrorWithSource(spec, err)
	}
	return nil
}

// ParseYAMLConfigString is like ParseYAMLConfig but for a string.
func ParseYAMLConfigString[T any](spec string, cfg T) error {
	return ParseYAMLConfig([]byte(spec), cfg)
}

// ParseYAMLConfigFile reads a yaml config file as per ParseYAMLConfig.
func ParseYAMLConfigFile[T any](file string, cfg T) error {
	spec, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if err := ParseYAMLConfig(spec, cfg); err != nil {
		return fmt.Errorf("failed to parse %s: %w", file, err)
	}
	return nil
}

func YAMLErrorWithSource(spec []byte, err error) error {
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
		fmt.Fprintf(&newError, "%vline %d: %q: %v", matches[1], l, specLines[l-1], matches[3])
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
