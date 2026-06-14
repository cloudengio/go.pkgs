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
	"os"
	"regexp"
	"strconv"
	"strings"

	"cloudeng.io/file"
	"cloudeng.io/file/localfs"
	"gopkg.in/yaml.v3"
)

// Option configures a Parser.
type Option func(*parserOptions)

// WithStrictFields causes Parse and ParseFiles to report an error for any YAML
// field that does not map to a struct field. Mapping fields at any level whose
// values carry a YAML anchor (&name) are permitted.
func WithStrictFields(strict bool) Option {
	return func(opts *parserOptions) {
		opts.strict = strict
	}
}

// WithYAMLVariables instructs the parser to collect scalar key-value pairs
// from the named top-level mapping and expand $VAR and ${VAR} references in
// specs before parsing.
func WithYAMLVariables(mapName string) Option {
	return func(opts *parserOptions) {
		opts.variablesMapName = mapName
	}
}

// WithExpandMapping expands ${VAR} and $VAR references in the spec
// using fn before parsing.
func WithExpandMapping(fn func(string) string) Option {
	return func(opts *parserOptions) {
		opts.expandEnv = fn
	}
}

// WithFS sets the file system used by ParseFiles. Defaults to the local OS
// file system rooted at the current working directory.
func WithFS(fs file.ReadFileFS) Option {
	return func(opts *parserOptions) {
		opts.fs = fs
	}
}

type parserOptions struct {
	strict           bool
	variablesMapName string
	fs               file.ReadFileFS
	expandEnv        func(string) string
}

// Parser parses and merges YAML configurations into a destination struct,
// optionally expanding environment variables and YAML-defined variables.
// Create one with NewParser.
type Parser struct {
	parserOptions
}

// NewParser returns a Parser configured with the supplied options.
func NewParser(opts ...Option) *Parser {
	p := &Parser{}
	for _, opt := range opts {
		opt(&p.parserOptions)
	}
	if p.fs == nil {
		p.fs = localfs.New()
	}
	return p
}

func (p *Parser) expandEnvPreserving(spec []byte) []byte {
	if p.expandEnv == nil {
		return spec
	}
	return []byte(os.Expand(string(spec), func(s string) string {
		rs := p.expandEnv(s)
		if len(rs) == 0 {
			return "${" + s + "}"
		}
		return rs
	}))
}

// Parse merges the YAML content of each spec into cfg. Specs are processed in
// order; a field present in a later spec overrides the value set by an earlier
// one, while fields only in an earlier spec are retained.
func (p *Parser) Parse(cfg any, specs ...[]byte) error {
	if len(specs) == 0 {
		return fmt.Errorf("no config specs provided")
	}
	ps := newParseState(p.strict, p.variablesMapName)
	for _, spec := range specs {
		spec = p.expandEnvPreserving(spec)
		if err := ps.parse("", spec, cfg); err != nil {
			return err
		}
	}
	return nil
}

// ParseFiles reads and merges the YAML contents of each named file into cfg.
// Files are processed in order; a field present in a later file overrides the
// value set by an earlier one, while fields only in an earlier file are
// retained. At least one filename must be supplied.
func (p *Parser) ParseFiles(ctx context.Context, cfg any, filenames ...string) error {
	if len(filenames) == 0 {
		return fmt.Errorf("no config files specified")
	}
	ps := newParseState(p.strict, p.variablesMapName)
	for _, filename := range filenames {
		if len(filename) == 0 {
			return fmt.Errorf("one of the filenames in %v is empty", filenames)
		}
		spec, err := p.fs.ReadFileCtx(ctx, filename)
		if err != nil {
			return fmt.Errorf("read %s: %w", filename, err)
		}
		spec = p.expandEnvPreserving(spec)
		if err := ps.parse(filename, spec, cfg); err != nil {
			return err
		}
	}
	return nil
}

// ParseConfigs merges the YAML content of each spec into cfg. Specs are
// processed in order; a field present in a later spec overrides the value set
// by an earlier one, while fields only in an earlier spec are retained.
func ParseConfigs(cfg any, specs ...[]byte) error {
	return NewParser().Parse(cfg, specs...)
}

// ParseConfigsStrict is like ParseConfigs but reports an error if there are
// unknown fields in the yaml specification. Mapping fields at any level
// whose values carry a YAML anchor (&name) are permitted.
func ParseConfigsStrict(cfg any, specs ...[]byte) error {
	return NewParser(WithStrictFields(true)).Parse(cfg, specs...)
}

// ParseConfigFiles reads and merges the YAML contents of each named file into
// cfg. Files are processed in order; a field present in a later file overrides
// the value set by an earlier one, while fields only in an earlier file are
// retained. At least one filename must be supplied.
func ParseConfigFiles(ctx context.Context, cfg any, filenames ...string) error {
	return NewParser().ParseFiles(ctx, cfg, filenames...)
}

// ParseConfigFilesStrict is like ParseConfigFiles but reports an error if any
// file contains unknown fields.
func ParseConfigFilesStrict(ctx context.Context, cfg any, filenames ...string) error {
	return NewParser(WithStrictFields(true)).ParseFiles(ctx, cfg, filenames...)
}

type parseState struct {
	strict           bool
	variablesMapName string
	anchors          map[string]anchorNode
	order            []string
	variables        *Variables
}

func newParseState(strict bool, variablesMapName string) *parseState {
	p := &parseState{
		strict:           strict,
		variablesMapName: variablesMapName,
		anchors:          map[string]anchorNode{},
		order:            []string{},
	}
	if variablesMapName != "" {
		p.variables = NewVariables()
	}
	return p
}

func (p *parseState) buildPreamble() ([]byte, error) {
	if len(p.anchors) == 0 {
		return nil, nil
	}
	content := make([]*yaml.Node, 0, 2*len(p.anchors))
	for _, name := range p.order {
		if an, ok := p.anchors[name]; ok {
			content = append(content, an.key, an.value)
		}
	}
	data, err := yaml.Marshal(&yaml.Node{Kind: yaml.MappingNode, Content: content})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p *parseState) withPreamble(spec []byte) ([]byte, int, error) {
	preamble, err := p.buildPreamble()
	if err != nil {
		return nil, 0, fmt.Errorf("internal error building YAML preamble: %w", err)
	}
	if len(preamble) == 0 {
		return spec, 0, nil
	}
	combined := append(append(preamble, preambleSep...), spec...)
	lineAdjustment := bytes.Count(preamble, []byte{'\n'}) + bytes.Count(preambleSep, []byte{'\n'})
	return combined, lineAdjustment, nil

}

var unknownFieldRE = regexp.MustCompile(`field\s+(\S+)\s+not\s+found\s+in\s+type`)
var preambleSep = []byte("\n---\n")

func (p *parseState) parse(filename string, spec []byte, cfg any) error {
	// Build the preamble from anchors accumulated by previous specs, then
	// register this spec's own anchors (for use by future specs and for
	// error filtering below).

	combined, lineAdjustment, err := p.withPreamble(spec)
	if err != nil {
		return err
	}
	p.preParse(combined)
	expanded := []byte(os.Expand(string(combined), p.variables.Mapping))
	dec := yaml.NewDecoder(bytes.NewReader(expanded))

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
		if len(p.anchors) == 0 && p.variablesMapName == "" {
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
				if fieldName == p.variablesMapName {
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

// preParse parses spec to:
//  1. collect the names and values of all mapping keys whose values carry YAML
//     anchors (&name) into p.anchors, these fields exist solely to define r
//     reusable anchors and are not themselves configuration and hence must
//     be ignored when parsing in strict mode.
//
// 2. collect a mapping keyed by p.variablesMapName of scalar key-value pairs
// from the named top-level mapping, if present, into p.variables.
func (p *parseState) preParse(spec []byte) {
	dec := yaml.NewDecoder(bytes.NewReader(spec))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			break // catches io.EOF
		}
		p.collectAnchorFields(&doc)
		if p.variables != nil {
			p.variables.mergeFromNode(&doc, p.variablesMapName)
		}
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
