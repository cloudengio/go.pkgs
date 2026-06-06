// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdjson_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdjson"
	"cloudeng.io/cmdutil/cmdyaml"
	"gopkg.in/yaml.v3"
)

type jsonTimeStruct struct {
	When     cmdjson.RFC3339Time `json:"when"`
	FlexTime cmdjson.FlexTime    `json:"flextime"`
}

type yamlTimeStruct struct {
	When     cmdyaml.RFC3339Time `yaml:"when"`
	FlexTime cmdyaml.FlexTime    `yaml:"flextime"`
}

func TestRFC3339Time(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	t.Run("Unmarshal", func(t *testing.T) {
		cfg := fmt.Sprintf(`{"when":%q}`, now.Format(time.RFC3339))
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(cfg), &s); err != nil {
			t.Fatal(err)
		}
		if got, want := time.Time(s.When), now; !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("Marshal", func(t *testing.T) {
		s := jsonTimeStruct{When: cmdjson.RFC3339Time(now)}
		data, err := json.Marshal(&s)
		if err != nil {
			t.Fatal(err)
		}
		var s2 jsonTimeStruct
		if err := json.Unmarshal(data, &s2); err != nil {
			t.Fatal(err)
		}
		if got, want := time.Time(s2.When), now; !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("UnmarshalNull", func(t *testing.T) {
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(`{"when":null}`), &s); err != nil {
			t.Fatalf("unexpected error for null: %v", err)
		}
		if !time.Time(s.When).IsZero() {
			t.Errorf("expected zero time for null, got %v", s.When)
		}
	})

	t.Run("UnmarshalInvalidTime", func(t *testing.T) {
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(`{"when":"not-a-time"}`), &s); err == nil {
			t.Error("expected error for invalid RFC3339 string")
		}
	})

	t.Run("UnmarshalNonString", func(t *testing.T) {
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(`{"when":12345}`), &s); err == nil {
			t.Error("expected error for non-string JSON value")
		}
	})

	t.Run("String", func(t *testing.T) {
		rt := cmdjson.RFC3339Time(now)
		if got, want := rt.String(), now.Format(time.RFC3339); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestFlexTime(t *testing.T) {
	tp := func(f, v string) time.Time {
		tt, err := time.Parse(f, v)
		if err != nil {
			t.Fatal(err)
		}
		return tt
	}

	for _, tc := range []struct {
		in     string
		format string
	}{
		{"2021-10-10", time.DateOnly},
		{"2021-10-10T03:03:03-07:00", time.RFC3339},
		{"03:03:05", time.TimeOnly},
		{"2021-10-10 03:03:05", time.DateTime},
	} {
		t.Run(tc.in, func(t *testing.T) {
			cfg := fmt.Sprintf(`{"flextime":%q}`, tc.in)
			var s jsonTimeStruct
			if err := json.Unmarshal([]byte(cfg), &s); err != nil {
				t.Fatalf("%v: %v", tc.in, err)
			}
			if got, want := time.Time(s.FlexTime), tp(tc.format, tc.in); !got.Equal(want) {
				t.Errorf("got %v, want %v", got, want)
			}
			t.Log(s.FlexTime.String())
		})
	}

	t.Run("MarshalIsRFC3339", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		s := jsonTimeStruct{FlexTime: cmdjson.FlexTime(now)}
		data, err := json.Marshal(&s)
		if err != nil {
			t.Fatal(err)
		}
		var raw struct {
			FlexTime string `json:"flextime"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		if _, err := time.Parse(time.RFC3339, raw.FlexTime); err != nil {
			t.Errorf("marshaled FlexTime %q is not RFC3339: %v", raw.FlexTime, err)
		}
	})

	t.Run("UnmarshalNull", func(t *testing.T) {
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(`{"flextime":null}`), &s); err != nil {
			t.Fatalf("unexpected error for null: %v", err)
		}
		if !time.Time(s.FlexTime).IsZero() {
			t.Errorf("expected zero time for null, got %v", s.FlexTime)
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(`{"flextime":"not-a-time"}`), &s); err == nil {
			t.Error("expected error for unrecognized time format")
		}
	})

	t.Run("UnmarshalNonString", func(t *testing.T) {
		var s jsonTimeStruct
		if err := json.Unmarshal([]byte(`{"flextime":12345}`), &s); err == nil {
			t.Error("expected error for non-string JSON value")
		}
	})
}

// TestJSONToYAMLRoundTrip verifies that times marshaled to JSON can be
// recovered after being expressed in YAML, and that the values survive the
// round-trip back to JSON.
//
// Both formats use RFC3339 as their canonical string representation, so a time
// string produced by MarshalJSON is a valid YAML scalar that UnmarshalYAML can
// parse.  The test exercises this compatibility path explicitly, which also
// reflects the common real-world scenario where a config document can be
// authored in either format.
func TestJSONToYAMLRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	// Marshal time values to JSON.
	jIn := jsonTimeStruct{
		When:     cmdjson.RFC3339Time(now),
		FlexTime: cmdjson.FlexTime(now),
	}
	jsonBytes, err := json.Marshal(&jIn)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Extract the raw RFC3339 strings that MarshalJSON produced.
	var rawJSON struct {
		When     string `json:"when"`
		FlexTime string `json:"flextime"`
	}
	if err := json.Unmarshal(jsonBytes, &rawJSON); err != nil {
		t.Fatalf("json.Unmarshal raw: %v", err)
	}

	// Embed those RFC3339 strings in a YAML document and unmarshal.  This is the
	// JSON→YAML step: the same string produced by JSON is valid YAML input.
	yamlStr := fmt.Sprintf("when: %s\nflextime: %s\n", rawJSON.When, rawJSON.FlexTime)
	var yMid yamlTimeStruct
	if err := yaml.Unmarshal([]byte(yamlStr), &yMid); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if got, want := time.Time(yMid.When), now; !got.Equal(want) {
		t.Errorf("when: got %v, want %v after JSON→YAML", got, want)
	}
	if got, want := time.Time(yMid.FlexTime), now; !got.Equal(want) {
		t.Errorf("flextime: got %v, want %v after JSON→YAML", got, want)
	}

	// Convert the YAML-recovered values back to JSON and compare to the
	// original JSON bytes — completing the JSON→YAML→JSON cycle.
	jOut := jsonTimeStruct{
		When:     cmdjson.RFC3339Time(time.Time(yMid.When)),
		FlexTime: cmdjson.FlexTime(time.Time(yMid.FlexTime)),
	}
	jsonBytes2, err := json.Marshal(&jOut)
	if err != nil {
		t.Fatalf("json.Marshal round-trip: %v", err)
	}
	if got, want := string(jsonBytes2), string(jsonBytes); got != want {
		t.Errorf("JSON→YAML→JSON mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

// TestYAMLToJSONRoundTrip verifies that times parsed from YAML can be
// recovered after being expressed in JSON, and that the values survive the
// round-trip back to YAML.
func TestYAMLToJSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	// Construct a YAML document from the canonical RFC3339 representation.
	rfc := now.Format(time.RFC3339)
	yamlStr := fmt.Sprintf("when: %s\nflextime: %s\n", rfc, rfc)

	// Parse the YAML.
	var yIn yamlTimeStruct
	if err := yaml.Unmarshal([]byte(yamlStr), &yIn); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	// Embed those time values in a JSON document.  This is the YAML→JSON step.
	jMid := jsonTimeStruct{
		When:     cmdjson.RFC3339Time(time.Time(yIn.When)),
		FlexTime: cmdjson.FlexTime(time.Time(yIn.FlexTime)),
	}
	jsonBytes, err := json.Marshal(&jMid)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Recover from JSON.
	var jOut jsonTimeStruct
	if err := json.Unmarshal(jsonBytes, &jOut); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if got, want := time.Time(jOut.When), now; !got.Equal(want) {
		t.Errorf("when: got %v, want %v after YAML→JSON", got, want)
	}
	if got, want := time.Time(jOut.FlexTime), now; !got.Equal(want) {
		t.Errorf("flextime: got %v, want %v after YAML→JSON", got, want)
	}

	// Verify the JSON-recovered time converts back to the same YAML string,
	// completing the YAML→JSON→YAML cycle.
	yamlStr2 := fmt.Sprintf("when: %s\nflextime: %s\n",
		jOut.When.String(), jOut.FlexTime.String())
	if got, want := yamlStr2, yamlStr; got != want {
		t.Errorf("YAML→JSON→YAML mismatch\ngot:  %s\nwant: %s", got, want)
	}
}
