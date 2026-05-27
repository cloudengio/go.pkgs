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

// ParseConfigStrict is like ParseConfig but reports an error if there
// are unknown fields in the yaml specification. Mapping fields at any level
// whose values carry a YAML anchor (&name) are permitted: they exist only
// to provide reusable values for alias references and are not struct fields.
//
// Deprecated: Use ParseConfigsStrict instead.
func ParseConfigStrict(spec []byte, cfg any) error {
	return parseConfigs(cfg, true, [][]byte{spec})
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

// ParseConfigsStrict is like ParseConfigs but reports an error if there are
// unknown fields in the yaml specification.  Mapping fields at any level
// whose values carry a YAML anchor (&name) are permitted: they exist only
// to provide reusable values for alias references and are not struct fields.
func ParseConfigsStrict(cfg any, specs ...[]byte) error {
	return parseConfigs(cfg, true, specs)
}

// ParseConfigString parses the yaml config in spec (as a string) into cfg.
func ParseConfigString(spec string, cfg any) error {
	return parseConfigs(cfg, false, [][]byte{[]byte(spec)})
}

// ParseConfigStringStrict is like ParseConfigString but reports an error if
// there are unknown fields in the yaml specification.
func ParseConfigStringStrict(spec string, cfg any) error {
	return parseConfigs(cfg, true, [][]byte{[]byte(spec)})
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
	order   []string
}

func newParseState(strict bool) *parseState {
	return &parseState{
		strict:  strict,
		anchors: map[string]anchorNode{},
		order:   []string{},
	}
}

func (p *parseState) buildPreamble() []byte {
	if len(p.anchors) == 0 {
		return nil
	}
	content := make([]*yaml.Node, 0, 2*len(p.anchors))
	for _, name := range p.order {
		if an, ok := p.anchors[name]; ok {
			content = append(content, an.key, an.value)
		}
	}
	data, _ := yaml.Marshal(&yaml.Node{Kind: yaml.MappingNode, Content: content})
	return data
}

var unknownFieldRE = regexp.MustCompile(`field\s+(\S+)\s+not\s+found\s+in\s+type`)
var preambleSep = []byte("\n---\n")

func (p *parseState) parse(filename string, spec []byte, cfg any) error {
	// Build the preamble from anchors accumulated by previous specs, then
	// register this spec's own anchors (for use by future specs and for
	// error filtering below).
	preamble := p.buildPreamble()
	p.allAnchorFields(spec)

	// Create a single decoder over a two-document stream:
	//   document 1 – preamble (anchor definitions from prior specs)
	//   document 2 – the actual spec
	// Decoding the preamble into a dummy yaml.Node registers its anchors in
	// the decoder's internal state so that alias references in the spec can
	// resolve, without touching cfg or triggering KnownFields errors.
	// The separator "\n---\n" contributes to line-number offsets reported by
	// the decoder; lineAdjustment corrects those back to spec-local line numbers.
	var dec *yaml.Decoder
	var lineAdjustment int
	if len(preamble) > 0 {
		combined := append(append(preamble, preambleSep...), spec...)
		lineAdjustment = bytes.Count(preamble, []byte{'\n'}) + bytes.Count(preambleSep, []byte{'\n'})
		dec = yaml.NewDecoder(bytes.NewReader(combined))
		var dummy yaml.Node
		if err := dec.Decode(&dummy); err != nil && !errors.Is(err, io.EOF) {
			return errorWithSource(filename, 0, spec, err)
		}
	} else {
		dec = yaml.NewDecoder(bytes.NewReader(spec))
	}

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
			return errorWithSource(filename, lineAdjustment, spec, err)
		}

		var yerr *yaml.TypeError
		if !errors.As(err, &yerr) {
			return errorWithSource(filename, lineAdjustment, spec, err)
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
			return errorWithSource(filename, lineAdjustment, spec, yerr)
		}
	}
	return nil
}

type anchorNode struct {
	key   *yaml.Node
	value *yaml.Node
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
			return err
		}
	}
	return nil
}

func parseConfigs(cfg any, strict bool, specs [][]byte) error {
	if len(specs) == 0 {
		return fmt.Errorf("no config specs provided")
	}
	ps := newParseState(strict)
	for _, spec := range specs {
		if err := ps.parse("", spec, cfg); err != nil {
			return err
		}
	}
	return nil
}

// allAnchorFields returns the names of all mapping keys in spec, at any
// nesting level, whose values carry a YAML anchor (&name). These fields exist
// solely to define reusable anchors and are not themselves configuration
// struct fields.
func (p *parseState) allAnchorFields(spec []byte) {
	dec := yaml.NewDecoder(bytes.NewReader(spec))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			break // catches io.EOF
		}
		p.collectAnchorFields(&doc)
	}
}

func (p *parseState) collectAnchorFields(node *yaml.Node) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			p.collectAnchorFields(child)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]
			if keyNode.Kind == yaml.ScalarNode && valNode.Anchor != "" {
				if _, ok := p.anchors[keyNode.Value]; !ok {
					p.order = append(p.order, keyNode.Value)
				}
				p.anchors[keyNode.Value] = anchorNode{key: keyNode, value: valNode}
			}
			p.collectAnchorFields(valNode)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			p.collectAnchorFields(child)
		}
		// AliasNode: skip — the anchor is recorded at its definition site.
	}
}

// errorWithSource returns an error that includes the yaml source
// code that was the cause of the error to help with debugging YAML
// errors.
// Note that the errors reported for the yaml parser may be inaccurate
// in terms of the lines the error is reported on. This seems to be particularly
// true for lists where errors with use of tabs to indent are often reported
// against the previous line rather than the offending one.
func errorWithSource(filename string, lineAdjustment int, spec []byte, err error) error {
	specLines := bytes.Split(spec, []byte{'\n'})
	if yerr, ok := err.(*yaml.TypeError); ok {
		return yamlTypeErrorWithSource(filename, lineAdjustment, specLines, yerr)
	}
	return yamlOtherErrorWithSource(filename, lineAdjustment, specLines, err)
}

var yamlPanicErrsRE = regexp.MustCompile(`(.*)line (\d+):\s*(.*)`)

func yamlOtherErrorWithSource(filename string, lineAdjustment int, specLines [][]byte, err error) error {
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
		l -= int64(lineAdjustment)
		if err != nil || l < 1 || int(l) > len(specLines) {
			newError.WriteString(errLine)
			newError.WriteRune('\n')
			continue
		}
		if filename != "" {
			fmt.Fprintf(&newError, "%s: line %d: %q: %v", filename, l, specLines[l-1], matches[3]) //nolint:gosec // G705: XSS via taint analysis not relevant here.
		} else {
			fmt.Fprintf(&newError, "%vline %d: %q: %v", matches[1], l, specLines[l-1], matches[3]) //nolint:gosec // G705: XSS via taint analysis not relevant here.
		}
	}
	return errors.New(newError.String())
}

var yamlTypeErrsRE = regexp.MustCompile(`\s*line (\d+):\s*(.*)`)

func yamlTypeErrorWithSource(filename string, lineAdjustment int, specLines [][]byte, err *yaml.TypeError) error {
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
		errLineNum := l - int64(lineAdjustment)
		if errLineNum < 1 || int(errLineNum) > len(specLines) {
			newErrors = append(newErrors, errLine)
			continue
		}
		if filename != "" {
			newErrors = append(newErrors, fmt.Sprintf("%s: line %d: %q: %v", filename, errLineNum, specLines[errLineNum-1], matches[2]))
		} else {
			newErrors = append(newErrors, fmt.Sprintf("line %d: %q: %v", errLineNum, specLines[errLineNum-1], matches[2]))
		}
	}
	return &yaml.TypeError{Errors: newErrors}
}
