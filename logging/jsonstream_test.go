// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package logging_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"cloudeng.io/logging"
)

func TestJSONFormatter(t *testing.T) {
	var buf bytes.Buffer
	jf := logging.NewJSONFormatter(&buf, "", "  ")

	// Test Write with valid JSON.
	validJSON := `{"a":1,"b":"two"}`
	n, err := jf.Write([]byte(validJSON))
	if err != nil {
		t.Errorf("Write with valid JSON failed: %v", err)
	}
	if got, want := n, len(validJSON); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	var m any
	if err := json.Unmarshal([]byte(validJSON), &m); err != nil {
		t.Fatalf("failed to unmarshal test json: %v", err)
	}
	pretty, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test json: %v", err)
	}
	// json.Encoder adds a newline.
	expected := string(pretty) + "\n"
	if got := buf.String(); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	// Test Write with invalid JSON.
	buf.Reset()
	invalidJSON := `{"a":1,"b":`
	_, err = jf.Write([]byte(invalidJSON))
	if err == nil {
		t.Errorf("expected an error for invalid json")
	}

	// Test Format.
	buf.Reset()
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{"test", 33}

	if err := jf.Format(data); err != nil {
		t.Errorf("Format failed: %v", err)
	}

	pretty, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal test json: %v", err)
	}
	expected = string(pretty) + "\n"
	if got := buf.String(); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	// Test with prefix and different indent.
	buf.Reset()
	jfWithPrefix := logging.NewJSONFormatter(&buf, "PREFIX", "\t")
	if err := jfWithPrefix.Format(data); err != nil {
		t.Errorf("Format failed: %v", err)
	}
	pretty, err = json.MarshalIndent(data, "PREFIX", "\t")
	if err != nil {
		t.Fatalf("failed to marshal test json: %v", err)
	}
	expected = string(pretty) + "\n"
	if got := buf.String(); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
