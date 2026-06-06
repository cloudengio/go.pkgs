// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml_test

import (
	"fmt"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdyaml"
	"gopkg.in/yaml.v3"
)

func TestMarshalYAML(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	want := now.Format(time.RFC3339)

	t.Run("RFC3339Time", func(t *testing.T) {
		rt := cmdyaml.RFC3339Time(now)
		val, err := rt.MarshalYAML()
		if err != nil {
			t.Fatal(err)
		}
		got, ok := val.(string)
		if !ok {
			t.Fatalf("expected string from MarshalYAML, got %T", val)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("FlexTime", func(t *testing.T) {
		ft := cmdyaml.FlexTime(now)
		val, err := ft.MarshalYAML()
		if err != nil {
			t.Fatal(err)
		}
		got, ok := val.(string)
		if !ok {
			t.Fatalf("expected string from MarshalYAML, got %T", val)
		}
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	// Verify that yaml.Marshal calls MarshalYAML when the field is a pointer.
	t.Run("MarshalViaPointerField", func(t *testing.T) {
		rt := cmdyaml.RFC3339Time(now)
		ft := cmdyaml.FlexTime(now)
		s := struct {
			When *cmdyaml.RFC3339Time `yaml:"when"`
			Flex *cmdyaml.FlexTime    `yaml:"flex"`
		}{When: &rt, Flex: &ft}
		data, err := yaml.Marshal(&s)
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			When *cmdyaml.RFC3339Time `yaml:"when"`
			Flex *cmdyaml.FlexTime    `yaml:"flex"`
		}
		if err := yaml.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		if !time.Time(*out.When).Equal(now) {
			t.Errorf("RFC3339Time: got %v, want %v", out.When, now)
		}
		if !time.Time(*out.Flex).Equal(now) {
			t.Errorf("FlexTime: got %v, want %v", out.Flex, now)
		}
	})
}

func TestTimeString(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	want := now.Format(time.RFC3339)

	if got := cmdyaml.RFC3339Time(now).String(); got != want {
		t.Errorf("RFC3339Time.String() got %q, want %q", got, want)
	}
	if got := cmdyaml.FlexTime(now).String(); got != want {
		t.Errorf("FlexTime.String() got %q, want %q", got, want)
	}
}

func TestTimeErrors(t *testing.T) {
	t.Run("RFC3339InvalidInput", func(t *testing.T) {
		var tt timeStruct
		if err := yaml.Unmarshal([]byte("when: not-a-time"), &tt); err == nil {
			t.Error("expected error for invalid RFC3339 input")
		}
	})

	t.Run("FlexTimeInvalidInput", func(t *testing.T) {
		var tt timeStruct
		if err := yaml.Unmarshal([]byte("flextime: not-a-time"), &tt); err == nil {
			t.Error("expected error for unrecognized time format")
		}
	})
}

type timeStruct struct {
	When     cmdyaml.RFC3339Time `yaml:"when"`
	FlexTime cmdyaml.FlexTime    `yaml:"flextime"`
}

func TestTime(t *testing.T) {
	tp := func(f, v string) time.Time {
		tv, err := time.Parse(f, v)
		if err != nil {
			t.Fatal(err)
		}
		return tv
	}

	var tt timeStruct
	now := time.Now().Truncate(time.Second)
	cfg := fmt.Sprintf("when: %v", now.Format(time.RFC3339))
	err := yaml.Unmarshal([]byte(cfg), &tt)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := time.Time(tt.When), now; !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for i, tc := range []struct {
		in     string
		format string
	}{
		{"2021-10-10", time.DateOnly},
		{"2021-10-10T03:03:03-07:00", time.RFC3339},
		{"03:03:05", time.TimeOnly},
		{"2021-10-10 03:03:05", time.DateTime},
	} {
		tt := &timeStruct{}
		cfg := fmt.Sprintf("flextime: %v", tc.in)
		err := yaml.Unmarshal([]byte(cfg), tt)
		if err != nil {
			t.Errorf("%v: %v", i, err)
		}

		if got, want := time.Time(tt.FlexTime), tp(tc.format, tc.in); !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
		t.Log(tt.FlexTime.String())

	}
}
