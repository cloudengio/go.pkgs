// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/subcmd"
)

const extensionSpec = `name: cli
summary: 
commands:
  - name: c1
    summary: c1 summary
    commands:
      - name: c1.1
        summary: c1.1
      {{range subcmdExtension "C1"}}
      {{.}}{{end}}

  {{range subcmdExtension "C2"}}{{.}}
  {{end}}

  - name: c3
    summary: c3 summary
    commands:
      {{range subcmdExtension "C3"}}{{.}}
      {{end}}
`

type extType struct {
	count int
}

func (e *extType) Set(_ *subcmd.CommandSetYAML) error {
	e.count++
	return nil
}

const (
	c1Spec = `
- name: c1.2
  summary: c1.2 summary
  commands:
    - name: c1.2.1
`

	c2Spec = `- name: c2
  summary: c2 summary
  commands:
    - name: c2.1
      summary: c2.1 summary
- name: c4
  commands:
    - name: c4.1
`
	c3Spec = `- name: c3.1
- name: c3.2
`

	expandedSpec = `name: cli
summary: 
commands:
  - name: c1
    summary: c1 summary
    commands:
      - name: c1.1
        summary: c1.1
      
      - name: c1.2
        summary: c1.2 summary
        commands:
          - name: c1.2.1

  - name: c2
    summary: c2 summary
    commands:
      - name: c2.1
        summary: c2.1 summary
  - name: c4
    commands:
      - name: c4.1
  

  - name: c3
    summary: c3 summary
    commands:
      - name: c3.1
      - name: c3.2
`
)

func TestExtensions(t *testing.T) {
	appendFunc := &extType{}
	c1 := subcmd.NewExtension("C1", c1Spec, appendFunc.Set)
	c2 := subcmd.NewExtension("C2", c2Spec, appendFunc.Set)
	c3 := subcmd.NewExtension("C3", c3Spec, appendFunc.Set)
	cs, expanded, err := subcmd.FromYAMLTemplate(extensionSpec, c1, c2, c3)
	if err != nil {
		t.Fatal(err)
	}
	if err := cs.AddExtensions(); err != nil {
		t.Fatal(err)
	}
	if got, want := appendFunc.count, 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := strings.TrimSpace(string(expanded)), strings.TrimSpace(expandedSpec); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

const (
	m1Spec = "- name: m1\n"
	m2Spec = "- name: m2\n"
	m3Spec = "- name: m3\n"

	mergedSpec = `name: cli
commands:
  {{range subcmdExtension "merged"}}{{.}}
  {{end}}
`
	expandedMergedSpec = `name: cli
commands:
  - name: m1
  - name: m2
  - name: m3
`
)

func TestMergeExtensions(t *testing.T) {
	appendFunc := &extType{}
	m1 := subcmd.NewExtension("m1", m1Spec, appendFunc.Set)
	m2 := subcmd.NewExtension("m2", m2Spec, appendFunc.Set)
	m3 := subcmd.NewExtension("m3", m3Spec, appendFunc.Set)
	merged := subcmd.MergeExtensions("merged", m1, m2, m3)
	cs, expanded, err := subcmd.FromYAMLTemplate(mergedSpec, merged)
	if err != nil {
		t.Fatal(err)
	}
	if err := cs.AddExtensions(); err != nil {
		t.Fatal(err)
	}
	if got, want := appendFunc.count, 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := strings.TrimSpace(string(expanded)), strings.TrimSpace(expandedMergedSpec); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
