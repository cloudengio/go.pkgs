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
	"io"
	"regexp"
	"strconv"
	"strings"

	"cloudeng.io/file"
	"gopkg.in/yaml.v3"
)

// ParseConfig will parse the yaml config in spec into the requested
// type. It provides improved error reporting via ErrorWithSource.
//
// Deprecated: Use ParseConfigs instead.
func ParseConfig(spec []byte, cfg any) error {
	return parseConfigs(cfg, false, [][]byte{spec})
}

// ParseConfigFile reads a yaml config file as per ParseConfig
// using file.FSReadFile to read the file. The use of FSReadFile allows
// for the configuration file to be read from storage system, including
// from embed.FS, instead of the local filesystem if an instance of fs.ReadFileFS
// is stored in the context.
//
// Deprecated: Use ParseConfigFiles instead.
func ParseConfigFile(ctx context.Context, filename string, cfg any) error {
	return parseConfigFiles(ctx, cfg, false, []string{filename})
}

// ParseConfigFileStrict is like ParseConfigFile but reports an error if there
// are unknown fields in the yaml specification.
//
// Deprecated: Use ParseConfigFilesStrict instead.
func ParseConfigFileStrict(ctx context.Context, filename string, cfg any) error {
	return parseConfigFiles(ctx, cfg, true, []string{filename})
}

// ParseConfigs merges the YAML content of each spec into cfg. Specs are
// processed in order; a field present in a later spec overrides the value set
// by an earlier one, while fields only in an earlier spec are retained.
func ParseConfigs(cfg any, specs ...[]byte) error {
	return parseConfigs(cfg, false, specs)
}

// ParseConfigsStrict is like ParseConfigs but reports an error if there are unknown fields in the yaml specification.
func ParseConfigsStrict(cfg any, specs ...[]byte) error {
	return parseConfigs(cfg, true, specs)
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

type parseState struct {
	strict  bool
	anchors map[string]anchorNode
}

func newParseState(strict bool) *parseState {
	return &parseState{
		strict:  strict,
		anchors: map[string]anchorNode{},
	}
}

var unknownFieldRE = regexp.MustCompile(`field\s+(\S+)\s+not\s+found\s+in\s+type`)

func (p *parseState) parse(filename string, spec []byte, cfg any) error {
	allAnchorFields(p.anchors, spec)
	dec := yaml.NewDecoder(bytes.NewReader(spec))
	if p.strict {
		dec.KnownFields(true)
	}
	for {
		err := dec.Decode(cfg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err == nil {
			continue
		}

		if len(p.anchors) == 0 {
			return errorWithSource(filename, spec, err)
		}

		yerr, ok := err.(*yaml.TypeError)
		if !ok {
			return errorWithSource(filename, spec, err)
		}

		var filtered []string
		for _, errStr := range yerr.Errors {
			matches := unknownFieldRE.FindStringSubmatch(errStr)
			if len(matches) == 2 {
				fieldName := matches[1]
				if _, ok := p.anchors[fieldName]; ok {
					continue
				}
			}
			filtered = append(filtered, errStr)
		}

		if len(filtered) > 0 {
			yerr.Errors = filtered
			return errorWithSource(filename, spec, yerr)
		}
	}
	return nil
}

type anchorNode struct {
	key   *yaml.Node
	value *yaml.Node
}

// ParseConfigStrict is like ParseConfig but reports an error if there
// are unknown fields in the yaml specification. Mapping fields at any level
// whose values carry a YAML anchor (&name) are permitted: they exist only
// to provide reusable values for alias references and are not struct fields.
func ParseConfigStrict(spec []byte, cfg any) error {
	return parseConfigs(cfg, true, [][]byte{spec})
	/*
		anchors := map[string]anchorNode{}

		return nil*/
}

func parseConfigFiles(ctx context.Context, cfg any, strict bool, filenames []string) error {
	if len(filenames) == 0 {
		return fmt.Errorf("no config files specified")
	}
	ps := newParseState(strict)
	for _, filename := range filenames {
		if len(filename) == 0 {
			return fmt.Errorf("one of the filenames in %v is empty", filenames)
		}
		spec, err := file.FSReadFile(ctx, filename)
		if err != nil {
			return fmt.Errorf("read %s: %w", filename, err)
		}
		if err := ps.parse(filename, spec, cfg); err != nil {
			return rewriteYAMLError(err, filenames, [][]byte{spec})
		}
	}
	return nil
	/*
		var err error
		if strict {
			err = ParseConfigsStrict(cfg, specs...)
		} else {
			err = ParseConfigs(cfg, specs...)
		}
		if err != nil {
			return rewriteYAMLError(err, filenames, specs)
		}
		return nil*/
}

func parseConfigs(cfg any, strict bool, specs [][]byte) error {
	if len(specs) == 0 {
		return fmt.Errorf("no config specs provided")
	}
	ps := newParseState(strict)
	for _, spec := range specs {
		if err := ps.parse("", spec, cfg); err != nil {
			return rewriteYAMLError(err, nil, specs)
		}
	}
	return nil
	/*
		var combined bytes.Buffer
		for i, spec := range specs {
			if i > 0 {
				combined.WriteString("\n---\n")
			}
			combined.Write(spec)
		}
		return parser(combined.Bytes(), cfg)*/

}

// allAnchorFields returns the names of all mapping keys in spec, at any
// nesting level, whose values carry a YAML anchor (&name). These fields exist
// solely to define reusable anchors and are not themselves configuration
// struct fields.
func allAnchorFields(fields map[string]anchorNode, spec []byte) map[string]anchorNode {
	dec := yaml.NewDecoder(bytes.NewReader(spec))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			break // catches io.EOF
		}
		collectAnchorFields(fields, &doc)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func collectAnchorFields(fields map[string]anchorNode, node *yaml.Node) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			collectAnchorFields(fields, child)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode && valNode.Anchor != "" {
				fields[keyNode.Value] = anchorNode{key: keyNode, value: valNode}
				fmt.Printf("found anchor %q for field %q\n", valNode.Anchor, keyNode.Value)
			}
			collectAnchorFields(fields, valNode)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			collectAnchorFields(fields, child)
		}
		// AliasNode: skip — the anchor is recorded at its definition site.
	}
}

func rewriteYAMLError(err error, filenames []string, specs [][]byte) error {
	lineOffsets := make([]int, len(specs))
	cumulativeLines := 0
	for i, spec := range specs {
		lineOffsets[i] = cumulativeLines
		if i > 0 {
			// separator is \n---\n
			cumulativeLines += 2
		}
		linesInSpec := bytes.Count(spec, []byte{'\n'})
		cumulativeLines += linesInSpec
	}

	findFile := func(globalLine int) (filename string, localLine int) {
		for i := len(lineOffsets) - 1; i >= 0; i-- {
			offset := lineOffsets[i]
			if i > 0 {
				offset += 2
			}
			if globalLine > offset {
				return filenames[i], globalLine - offset
			}
		}
		if len(filenames) > 0 {
			return filenames[0], globalLine
		}
		return "", globalLine
	}

	var yerr *yaml.TypeError
	if errors.As(err, &yerr) {
		re := regexp.MustCompile(`line (\d+): (.*)`)
		newErrors := make([]string, len(yerr.Errors))
		for i, errStr := range yerr.Errors {
			matches := re.FindStringSubmatch(errStr)
			if len(matches) != 3 {
				newErrors[i] = errStr
				continue
			}
			globalLine, _ := strconv.Atoi(matches[1])
			filename, localLine := findFile(globalLine)
			newErrors[i] = fmt.Sprintf("%s: line %d: %s", filename, localLine, matches[2])
		}
		yerr.Errors = newErrors
		return fmt.Errorf("failed to parse config files: %w", yerr)
	}
	// TODO(cnicolaou): handle non-TypeError errors which are returned as generic
	// errors and would require string matching on err.Error().
	return fmt.Errorf("failed to parse config files: %w", err)
}

/*
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
}*/

// errorWithSource returns an error that includes the yaml source
// code that was the cause of the error to help with debugging YAML
// errors.
// Note that the errors reported for the yaml parser may be inaccurate
// in terms of the lines the error is reported on. This seems to be particularly
// true for lists where errors with use of tabs to indent are often reported
// against the previous line rather than the offending one.
func errorWithSource(filename string, spec []byte, err error) error {
	specLines := bytes.Split(spec, []byte{'\n'})
	if yerr, ok := err.(*yaml.TypeError); ok {
		return yamlTypeErrorWithSource(filename, specLines, yerr)
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

func yamlTypeErrorWithSource(filename string, specLines [][]byte, err *yaml.TypeError) error {
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
		if filename != "" {
			newErrors = append(newErrors, fmt.Sprintf("%s: line %d: %q: %v", filename, l, specLines[l-1], matches[2]))
		} else {
			newErrors = append(newErrors, fmt.Sprintf("line %d: %q: %v", l, specLines[l-1], matches[2]))
		}
	}
	return &yaml.TypeError{Errors: newErrors}
}
