// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cicd

import (
	"fmt"
	"testing"
)

// mockT captures calls to Skipf so tests can assert on skip behaviour without
// actually skipping the enclosing test.
type mockT struct {
	name      string
	skipMsg   string
	skipped   bool
	helpCalls int
}

func (m *mockT) Helper()                  { m.helpCalls++ }
func (m *mockT) Skipf(f string, a ...any) { m.skipped = true; m.skipMsg = fmt.Sprintf(f, a...) }
func (m *mockT) Name() string             { return m.name }
func (m *mockT) Fatalf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}

func TestSkipIf(t *testing.T) {
	m := &mockT{name: "TestSkipIf"}
	SkipIf(m, "should skip", true)
	if !m.skipped {
		t.Error("expected skip when cond is true")
	}
	if m.skipMsg != "should skip" {
		t.Errorf("unexpected skip message: %q", m.skipMsg)
	}

	m2 := &mockT{name: "TestSkipIf"}
	SkipIf(m2, "should not skip", false)
	if m2.skipped {
		t.Error("expected no skip when cond is false")
	}
}
