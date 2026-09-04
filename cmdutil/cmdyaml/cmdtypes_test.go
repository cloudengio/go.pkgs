// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

// The types in cmdtypes are encoded and decoded through their MarshalText and
// UnmarshalText methods, which yaml.v3 uses, so cmdtypes itself does not
// depend on yaml. Their YAML behaviour is verified here instead.

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdtypes"
	"gopkg.in/yaml.v3"
)

func TestCmdtypesPermissionsYAML(t *testing.T) {
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
		// The special bits are the same value whichever notation is used.
		{"perm: 4755\n", 0755 | fs.ModeSetuid},
		{"perm: \"rwsr-xr-x\"\n", 0755 | fs.ModeSetuid},
		{"perm: 1777\n", 0777 | fs.ModeSticky},
		{"perm: \"rwxrwxrwt\"\n", 0777 | fs.ModeSticky},
	} {
		var got spec
		if err := yaml.Unmarshal([]byte(tc.yaml), &got); err != nil {
			t.Errorf("%q: %v", tc.yaml, err)
			continue
		}
		if got, want := got.Perm.FileMode(), tc.want; got != want {
			t.Errorf("%q: got %v, want %v", tc.yaml, got, want)
		}
	}
}

func TestCmdtypesPermissionsYAMLMarshal(t *testing.T) {
	type spec struct {
		Perm cmdtypes.Permissions `yaml:"perm"`
	}
	for _, tc := range []struct {
		perm cmdtypes.Permissions
		want string
	}{
		{cmdtypes.Permissions(0755), "perm: \"0755\"\n"},
		{cmdtypes.Permissions(0755 | fs.ModeSetuid), "perm: \"4755\"\n"},
		{cmdtypes.Permissions(0777 | fs.ModeSticky), "perm: \"1777\"\n"},
	} {
		buf, err := yaml.Marshal(spec{Perm: tc.perm})
		if err != nil {
			t.Errorf("%v: %v", tc.perm, err)
			continue
		}
		if got := string(buf); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
		// It reads back as the same value, which the four digit form is
		// chosen to guarantee.
		var back spec
		if err := yaml.Unmarshal(buf, &back); err != nil {
			t.Errorf("%v: %v", tc.perm, err)
			continue
		}
		if back.Perm != tc.perm {
			t.Errorf("got %v, want %v", back.Perm.FileMode(), tc.perm.FileMode())
		}
	}
}

// TestCmdtypesPermissionsBothEncodings verifies that JSON and YAML agree, so
// that the same configuration can be expressed in either.
func TestCmdtypesPermissionsBothEncodings(t *testing.T) {
	for _, in := range []string{"0700", "700", "rwxr-xr-x", "u=rwx,go=", "rw-r--r--", "4755", "rwsr-xr-x"} {
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

func TestCmdtypesPermissionsYAMLErrors(t *testing.T) {
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
			t.Errorf("%q: expected an error, got %v", in, got.Perm.FileMode())
		}
	}
}

func TestCmdtypesFlexTimeYAML(t *testing.T) {
	for _, tc := range []struct {
		in     string
		format string
	}{
		{"2021-10-10", time.DateOnly},
		{"2021-10-10T03:03:03-07:00", time.RFC3339},
		{"03:03:05", time.TimeOnly},
		{"2021-10-10 03:03:05", time.DateTime},
	} {
		want, err := time.Parse(tc.format, tc.in)
		if err != nil {
			t.Fatal(err)
		}
		var got cmdtypes.FlexTime
		if err := yaml.Unmarshal([]byte(fmt.Sprintf("%q", tc.in)), &got); err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if !got.Time().Equal(want) {
			t.Errorf("%v: got %v, want %v", tc.in, got.Time(), want)
		}

		// Whichever format it was read from, it is written as RFC3339 and
		// reads back as the same instant.
		buf, err := yaml.Marshal(got)
		if err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		var back cmdtypes.FlexTime
		if err := yaml.Unmarshal(buf, &back); err != nil {
			t.Errorf("%v: %v", tc.in, err)
			continue
		}
		if !back.Time().Equal(got.Time()) {
			t.Errorf("%v: got %v, want %v", tc.in, back.Time(), got.Time())
		}
	}
}

func TestCmdtypesFlexTimeYAMLErrors(t *testing.T) {
	var ft cmdtypes.FlexTime
	if err := yaml.Unmarshal([]byte("not-a-time\n"), &ft); err == nil {
		t.Error("expected an error for an unrecognised format")
	}
	// A null leaves the value alone rather than reporting an error.
	base, err := cmdtypes.ParseFlexTime("2021-10-10T03:03:03Z")
	if err != nil {
		t.Fatal(err)
	}
	got := base
	if err := yaml.Unmarshal([]byte("null\n"), &got); err != nil {
		t.Fatalf("null: %v", err)
	}
	if !got.Time().Equal(base.Time()) {
		t.Errorf("null: got %v, want the value to be left alone", got.Time())
	}
}
