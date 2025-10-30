// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package logging

import (
	"encoding/json"
	"io"
)

type JSONFormatter struct {
	writer io.Writer
	prefix string
	indent string
}

func NewJSONFormatter(w io.Writer, prefix, indent string) *JSONFormatter {
	return &JSONFormatter{
		writer: w,
		prefix: prefix,
		indent: indent,
	}
}

func (js *JSONFormatter) Write(p []byte) (n int, err error) {
	err = js.Format(json.RawMessage(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (js *JSONFormatter) Format(v any) error {
	enc := json.NewEncoder(js.writer)
	enc.SetIndent(js.prefix, js.indent)
	return enc.Encode(v)
}
