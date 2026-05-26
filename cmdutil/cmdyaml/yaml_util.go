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
	"reflect"
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
//
// Deprecated: Use ParseConfigFiles instead.
func ParseConfigFile(ctx context.Context, filename string, cfg any) error {
	return parseConfigFile(ctx, filename, cfg, ParseConfig)
}

// ParseConfigStrict is like ParseConfig but reports an error if there
// are unknown fields in the yaml specification. Top-level mapping fields
// whose values carry a YAML anchor (&name) are permitted: they exist only
// to provide reusable values for alias references and are not struct fields.
func ParseConfigStrict(spec []byte, cfg any) error {
	anchors := topLevelAnchorFields(spec)
	if len(anchors) == 0 {
		dec := yaml.NewDecoder(bytes.NewReader(spec))
		dec.KnownFields(true)
		if err := dec.Decode(cfg); err != nil {
			return ErrorWithSource(spec, err)
		}
		return nil
	}
	// Build cleaned YAML: expand all alias references into a generic map,
	// remove anchor-definition fields, re-marshal.
	var raw map[string]any
	if err := yaml.Unmarshal(spec, &raw); err != nil {
		return ErrorWithSource(spec, err)
	}
	for field := range anchors {
		delete(raw, field)
	}
	cleaned, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	// Strict field-name check first: decode cleaned into a zero-value dummy of
	// the same type so unknown fields are caught before cfg is touched.
	// Error line numbers reference cleaned (not spec) because the marshal
	// round-trip may reorder or expand flow nodes.
	dummy := reflect.New(reflect.TypeOf(cfg).Elem()).Interface()
	dec := yaml.NewDecoder(bytes.NewReader(cleaned))
	dec.KnownFields(true)
	if err := dec.Decode(dummy); err != nil {
		return ErrorWithSource(cleaned, err)
	}
	// Strict check passed: decode spec faithfully into cfg, preserving all
	// custom types (Deferred, ByteSize, …) via their UnmarshalYAML methods.
	if err := yaml.Unmarshal(spec, cfg); err != nil {
		return ErrorWithSource(spec, err)
	}
	return nil
}

// topLevelAnchorFields returns the names of top-level mapping keys in spec
// whose values carry a YAML anchor (&name). These fields exist solely to
// define reusable anchors and are not themselves configuration struct fields.
func topLevelAnchorFields(spec []byte) map[string]bool {
	var doc yaml.Node
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		return nil
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil
	}
	fields := map[string]bool{}
	mapping := doc.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valNode := mapping.Content[i+1]
		if keyNode.Kind == yaml.ScalarNode && valNode.Anchor != "" {
			fields[keyNode.Value] = true
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// ParseConfigStringStrict is like ParseConfigString but reports an error if there
// are unknown fields in the yaml specification.
func ParseConfigStringStrict(spec string, cfg any) error {
	return ParseConfigStrict([]byte(spec), cfg)
}

// ParseConfigFileStrict is like ParseConfigFile but reports an error if there
// are unknown fields in the yaml specification.
//
// Deprecated: Use ParseConfigFilesStrict instead.
func ParseConfigFileStrict(ctx context.Context, filename string, cfg any) error {
	return parseConfigFile(ctx, filename, cfg, ParseConfigStrict)
}

// ParseConfigs merges the YAML content of each spec into cfg. Specs are
// processed in order; a field present in a later spec overrides the value set
// by an earlier one, while fields only in an earlier spec are retained.
func ParseConfigs(cfg any, specs ...[]byte) error {
	return parseConfigs(cfg, ParseConfig, specs)
}

// ParseConfigsStrict is like ParseConfigs but reports an error if any spec
// contains unknown fields.
func ParseConfigsStrict(cfg any, specs ...[]byte) error {
	return parseConfigs(cfg, ParseConfigStrict, specs)
}

// ParseConfigFiles reads and merges the YAML contents of each named file into
// cfg. Files are processed in order; a field present in a later file overrides
// the value set by an earlier one, while fields only in an earlier file are
// retained. At least one filename must be supplied.
func ParseConfigFiles(ctx context.Context, cfg any, filenames ...string) error {
	return parseConfigFiles(ctx, cfg, false, filenames)
}

// ParseConfigFilesStrict is like ParseConfigFiles but reports an error if any
// file contains unknown fields.
func ParseConfigFilesStrict(ctx context.Context, cfg any, filenames ...string) error {
	return parseConfigFiles(ctx, cfg, true, filenames)
}

func parseConfigFiles(ctx context.Context, cfg any, strict bool, filenames []string) error {
	if len(filenames) == 0 {
		return fmt.Errorf("no config files specified")
	}
	for _, filename := range filenames {
		if len(filename) == 0 {
			return fmt.Errorf("one of the filenames in %v is empty", filenames)
		}
		data, err := file.FSReadFile(ctx, filename)
		if err != nil {
			return fmt.Errorf("read %s: %w", filename, err)
		}
		if strict {
			err = ParseConfigStrict(data, cfg)
		} else {
			err = ParseConfig(data, cfg)
		}
		if err != nil {
			return fmt.Errorf("failed to parse: %q: %w", filename, err)
		}
	}
	return nil
}

func parseConfigs(cfg any, parser func([]byte, any) error, specs [][]byte) error {
	for _, spec := range specs {
		if err := parser(spec, cfg); err != nil {
			return err
		}
	}
	return nil
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
		if err != nil || l < 1 || int(l) > len(specLines) {
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
		if err != nil || l < 1 || int(l) > len(specLines) {
			newErrors = append(newErrors, errLine)
			continue
		}
		newErrors = append(newErrors, fmt.Sprintf("line %d: %q: %v", l, specLines[l-1], matches[2]))
	}
	return &yaml.TypeError{Errors: newErrors}
}
