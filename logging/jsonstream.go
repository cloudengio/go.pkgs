// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package logging

import (
	"encoding/json"
	"io"
)

// JSONFormatter implements a log formatter that outputs JSON formatted logs.
type JSONFormatter struct {
	enc *json.Encoder
}

// NewJSONFormatter creates a new JSONFormatter that writes to with the specified
// prefix and indent.
func NewJSONFormatter(w io.Writer, prefix, indent string) *JSONFormatter {
	enc := json.NewEncoder(w)
	enc.SetIndent(prefix, indent)
	return &JSONFormatter{enc: enc}
}

// Write implements the io.Writer interface and it assumes that it is
// called with a complete JSON object.
func (js *JSONFormatter) Write(p []byte) (n int, err error) {
	err = js.Format(json.RawMessage(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Format formats the specified value as JSON.
func (js *JSONFormatter) Format(v any) error {
	return js.enc.Encode(v)
}
