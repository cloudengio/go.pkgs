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
