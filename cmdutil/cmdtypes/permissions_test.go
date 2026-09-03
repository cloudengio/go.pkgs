// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdtypes_test

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/cmdtypes"
	"gopkg.in/yaml.v3"
)

func TestPermissionsUnmarshalJSON(t *testing.T) {
	type spec struct {
		Perm cmdtypes.Permissions `json:"perm"`
	}

	for _, tc := range []struct {
		json string
		want fs.FileMode
	}{
		// Strings, in each of the notations the parser accepts.
		{`{"perm":"0700"}`, 0700},
		{`{"perm":"700"}`, 0700},
		{`{"perm":"0o700"}`, 0700},
		{`{"perm":"0x1c0"}`, 0700},
		{`{"perm":"rwx------"}`, 0700},
		{`{"perm":"-rwx------"}`, 0700},
		{`{"perm":"rwx"}`, 0700},
		{`{"perm":"u=rwx,go="}`, 0700},
		{`{"perm":"rwxr-xr-x"}`, 0755},
		{`{"perm":"0644"}`, 0644},
		{`{"perm":"u=rwx,go=rx"}`, 0755},

		// A number is read as octal, as it is in YAML, so that 700 does not
		// silently mean decimal 700.
		{`{"perm":700}`, 0700},
		{`{"perm":755}`, 0755},
		{`{"perm":644}`, 0644},
	} {
		var got spec
		if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
			t.Errorf("%v: %v", tc.json, err)
			continue
		}
		if got.Perm.FileMode() != tc.want {
			t.Errorf("%v: got %04o, want %04o", tc.json, got.Perm.FileMode(), tc.want)
		}
	}
}

// TestPermissionsNumberIsOctal makes the interpretation of a bare number
// explicit, since reading it as decimal would silently grant the wrong bits.
func TestPermissionsNumberIsOctal(t *testing.T) {
	var p cmdtypes.Permissions
	if err := json.Unmarshal([]byte(`700`), &p); err != nil {
		t.Fatal(err)
	}
	if got, want := p.FileMode(), fs.FileMode(0700); got != want {
		t.Errorf("got %04o, want %04o", got, want)
	}
	if got := p.FileMode(); got == fs.FileMode(700) {
		t.Errorf("got %v, the decimal reading of 700", got)
	}
}

func TestPermissionsMarshalJSON(t *testing.T) {
	type spec struct {
		Perm cmdtypes.Permissions `json:"perm"`
	}
	for _, tc := range []struct {
		perm fs.FileMode
		want string
	}{
		{0700, `{"perm":"0700"}`},
		{0755, `{"perm":"0755"}`},
		{0644, `{"perm":"0644"}`},
		{0, `{"perm":"0000"}`},
	} {
		buf, err := json.Marshal(spec{Perm: cmdtypes.Permissions(tc.perm)})
		if err != nil {
			t.Errorf("%04o: %v", tc.perm, err)
			continue
		}
		if got := string(buf); got != tc.want {
			t.Errorf("%04o: got %v, want %v", tc.perm, got, tc.want)
		}
	}
}

// TestPermissionsRoundTrip verifies that what is marshaled reads back
// unchanged, whatever notation it arrived in.
func TestPermissionsRoundTrip(t *testing.T) {
	for _, in := range []string{"0700", "700", "rwxr-xr-x", "u=rwx,go=", "0644", "rwx"} {
		p, err := cmdtypes.ParsePermissions(in)
		if err != nil {
			t.Errorf("%v: %v", in, err)
			continue
		}
		buf, err := json.Marshal(p)
		if err != nil {
			t.Errorf("%v: %v", in, err)
			continue
		}
		var got cmdtypes.Permissions
		if err := json.Unmarshal(buf, &got); err != nil {
			t.Errorf("%v: %v", in, err)
			continue
		}
		if got != p {
			t.Errorf("%v: got %v, want %v", in, got, p)
		}
	}
}

func TestPermissionsErrors(t *testing.T) {
	for _, tc := range []struct {
		json, want string
	}{
		{`{"perm":"not-a-permission"}`, "invalid permissions format"},
		{`{"perm":"999"}`, "invalid permissions format"},
		{`{"perm":""}`, "empty permissions"},
		{`{"perm":true}`, "expected a string or number"},
		{`{"perm":[]}`, "expected a string or number"},
		{`{"perm":{}}`, "expected a string or number"},
	} {
		type spec struct {
			Perm cmdtypes.Permissions `json:"perm"`
		}
		var got spec
		err := json.Unmarshal([]byte(tc.json), &got)
		if err == nil {
			t.Errorf("%v: expected an error, got %04o", tc.json, got.Perm.FileMode())
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want it to contain %q", tc.json, err, tc.want)
		}
	}
}

// TestPermissionsNull verifies that a null leaves the value untouched rather
// than zeroing it, matching the other types in this package.
func TestPermissionsNull(t *testing.T) {
	p := cmdtypes.Permissions(0755)
	if err := json.Unmarshal([]byte(`null`), &p); err != nil {
		t.Fatal(err)
	}
	if got, want := p.FileMode(), fs.FileMode(0755); got != want {
		t.Errorf("got %04o, want %04o", got, want)
	}
}

func TestPermissionsText(t *testing.T) {
	var p cmdtypes.Permissions
	if err := p.UnmarshalText([]byte("u=rwx,go=rx")); err != nil {
		t.Fatal(err)
	}
	if got, want := p.FileMode(), fs.FileMode(0755); got != want {
		t.Errorf("got %04o, want %04o", got, want)
	}
	text, err := p.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(text), "0755"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := p.String(), "0755"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestPermissionsYAML verifies that the same type decodes from YAML, which
// routes through UnmarshalText, including unquoted numbers such as 0700 that
// YAML presents as integer nodes.
func TestPermissionsYAML(t *testing.T) {
	type spec struct {
		Perm cmdtypes.Permissions `yaml:"perm"`
	}
	for _, tc := range []struct {
		yaml string
		want fs.FileMode
	}{
		{"perm: 0700\n", 0700},
		{"perm: \"0700\"\n", 0700},
		{"perm: 700\n", 0700},
		{"perm: 0o700\n", 0700},
		{"perm: \"rwx------\"\n", 0700},
		{"perm: \"-rwx------\"\n", 0700},
		{"perm: \"u=rwx,go=\"\n", 0700},
		{"perm: 0755\n", 0755},
		{"perm: \"rwxr-xr-x\"\n", 0755},
		{"perm: \"u=rwx,go=rx\"\n", 0755},
		{"perm: \"rw-r--r--\"\n", 0644},
	} {
		var got spec
		if err := yaml.Unmarshal([]byte(tc.yaml), &got); err != nil {
			t.Errorf("%q: %v", tc.yaml, err)
			continue
		}
		if got, want := got.Perm.FileMode(), tc.want; got != want {
			t.Errorf("%q: got %04o, want %04o", tc.yaml, got, want)
		}
	}
}

func TestPermissionsYAMLMarshal(t *testing.T) {
	type spec struct {
		Perm cmdtypes.Permissions `yaml:"perm"`
	}
	buf, err := yaml.Marshal(spec{Perm: cmdtypes.Permissions(0755)})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(buf), "perm: \"0755\"\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// It reads back unchanged.
	var got spec
	if err := yaml.Unmarshal(buf, &got); err != nil {
		t.Fatal(err)
	}
	if got, want := got.Perm.FileMode(), fs.FileMode(0755); got != want {
		t.Errorf("got %04o, want %04o", got, want)
	}
}

// TestPermissionsSameValueBothEncodings verifies that JSON and YAML agree, so
// that the same configuration can be expressed in either.
func TestPermissionsSameValueBothEncodings(t *testing.T) {
	for _, in := range []string{"0700", "700", "rwxr-xr-x", "u=rwx,go=", "rw-r--r--"} {
		want, err := cmdtypes.ParsePermissions(in)
		if err != nil {
			t.Errorf("%v: %v", in, err)
			continue
		}
		var fromJSON, fromYAML cmdtypes.Permissions
		if err := json.Unmarshal([]byte(`"`+in+`"`), &fromJSON); err != nil {
			t.Errorf("%v: json: %v", in, err)
			continue
		}
		if err := yaml.Unmarshal([]byte(`"`+in+`"`), &fromYAML); err != nil {
			t.Errorf("%v: yaml: %v", in, err)
			continue
		}
		if fromJSON != want || fromYAML != want {
			t.Errorf("%v: json gave %v, yaml gave %v, want %v", in, fromJSON, fromYAML, want)
		}
	}
}

// TestPermissionsYAMLErrors verifies that an unusable value is reported rather
// than silently left at zero, which would read as no permissions at all.
func TestPermissionsYAMLErrors(t *testing.T) {
	type spec struct {
		Perm cmdtypes.Permissions `yaml:"perm"`
	}
	for _, in := range []string{
		"perm: not-a-permission\n",
		"perm: \"999\"\n",
		"perm: []\n",
		"perm: {a: b}\n",
	} {
		var got spec
		if err := yaml.Unmarshal([]byte(in), &got); err == nil {
			t.Errorf("%q: expected an error, got %04o", in, got.Perm.FileMode())
		}
	}
}
