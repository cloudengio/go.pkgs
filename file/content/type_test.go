// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content_test

import (
	"testing"

	"cloudeng.io/file/content"
)

func TestContentTypes(t *testing.T) {
	for _, tc := range []struct {
		input           content.Type
		typ, par, value string
	}{
		{"text/plain", "text/plain", "", ""},
		{"text/plain ", "text/plain", "", ""},
		{"text/html; charset=utf8", "text/html", "charset", "utf8"},
		{"text/html;charset= utf8", "text/html", "charset", "utf8"},
		{"text/html ;charset= utf8", "text/html", "charset", "utf8"},
		{"text/html;charset=utf8", "text/html", "charset", "utf8"},
	} {
		typ, par, value, err := content.ParseTypeFull(tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if got, want := typ, tc.typ; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}

		if got, want := par, tc.par; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
		if got, want := value, tc.value; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}

		typ, err = content.ParseType(tc.input)
		if err != nil {
			t.Errorf("%v: %v", tc.input, err)
			continue
		}
		if got, want := typ, tc.typ; got != want {
			t.Errorf("%v: got %v, want %v", tc.input, got, want)
		}
	}

	_, _, _, err := content.ParseTypeFull("text/html;charset=utf8;something=else")
	if err == nil || err.Error() != "invalid parameter value: text/html;charset=utf8;something=else" {
		t.Errorf("missing or incorrect error: %v", err)
	}

	_, _, _, err = content.ParseTypeFull("text/html/html;charset=utf8")
	if err == nil || err.Error() != "invalid content type: text/html/html;charset=utf8" {
		t.Errorf("missing or incorrect error: %v", err)
	}
}
