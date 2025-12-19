// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdyaml"
)

type testStruct struct {
	Field []int
}

func TestYAMLErrors(t *testing.T) {
	var ts testStruct
	for i, tc := range []struct {
		input, errMsg string
	}{
		{`xxx: - err`, "yaml: block sequence entries are not allowed in this context"},

		{`
xxx: - err
`, `yaml: line 2: "xxx: - err": block sequence entries are not allowed in this context`},

		{`
	tab: 2`, `yaml: line 2: "\ttab: 2": found character that cannot start any token`},

		{`notab: 2
	tab: 3`, `yaml: line 2: "\ttab: 3": found a tab character that violates indentation`},

		{`	notab: 2`, `yaml: found character that cannot start any token`},

		{`

	tab: 2`, `yaml: line 3: "\ttab: 2": found character that cannot start any token`},

		{`
field:
  ts1: [1,2]`, "yaml: unmarshal errors:\n" + `  line 3: "  ts1: [1,2]": cannot unmarshal !!map into []int`},

		// Note that the yaml parser does not always get the line number correct!
		// It seems to be wrong for lists in particular.
		{`
list:
  - a
	  - b
`, `yaml: line 3: "  - a": found a tab character that violates indentation`},
	} {
		err := cmdyaml.ParseConfigString(tc.input, &ts)
		if err == nil || strings.TrimSpace(err.Error()) != tc.errMsg {
			t.Errorf("%v: got %v, want %v", i, err, tc.errMsg)
		}
	}
}

func TestStrictParse(t *testing.T) {
	var ts testStruct
	input := `
field: [1,2]
unknown: [3,4]
`
	err := cmdyaml.ParseConfigString(input, &ts)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = cmdyaml.ParseConfigStringStrict(input, &ts)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if got, want := err.Error(), `line 3: "unknown: [3,4]": field unknown not found in type cmdyaml_test.testStruct`; !strings.Contains(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
