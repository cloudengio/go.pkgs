// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdjson_test

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"testing"
	"time"

	"cloudeng.io/cmdutil/cmdjson"
	"cloudeng.io/cmdutil/cmdtypes"
	"cloudeng.io/cmdutil/cmdyaml"
	"gopkg.in/yaml.v3"
)

type jsonTimeStruct struct {
	When     cmdjson.RFC3339Time `json:"when"`
	FlexTime cmdtypes.FlexTime   `json:"flextime"`
}

type yamlTimeStruct struct {
	When     cmdyaml.RFC3339Time `yaml:"when"`
	FlexTime cmdtypes.FlexTime   `yaml:"flextime"`
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
		FlexTime: cmdtypes.FlexTime(now),
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
		FlexTime: cmdtypes.FlexTime(time.Time(yMid.FlexTime)),
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
		FlexTime: cmdtypes.FlexTime(time.Time(yIn.FlexTime)),
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

// TestRFC3339TimeJSONV2 verifies the encoding/json/v2 methods, which write to
// and read from the token stream directly rather than through an intermediate
// value. json/v2 prefers MarshalJSONTo and UnmarshalJSONFrom over the v1
// methods, so these are what it uses.
func TestRFC3339TimeJSONV2(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	rt := cmdjson.RFC3339Time(now)

	// The v2 encoding is the same as the v1 encoding.
	v2buf, err := jsonv2.Marshal(rt)
	if err != nil {
		t.Fatalf("json/v2 marshal: %v", err)
	}
	v1buf, err := json.Marshal(rt)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	if got, want := string(v2buf), string(v1buf); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := string(v2buf), fmt.Sprintf("%q", now.Format(time.RFC3339)); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// It reads back what either encoder wrote.
	for _, buf := range [][]byte{v1buf, v2buf} {
		var back cmdjson.RFC3339Time
		if err := jsonv2.Unmarshal(buf, &back); err != nil {
			t.Errorf("%s: %v", buf, err)
			continue
		}
		if got := time.Time(back); !got.Equal(now) {
			t.Errorf("%s: got %v, want %v", buf, got, now)
		}
	}

	// A null leaves the value unchanged, as it does for UnmarshalJSON.
	unchanged := cmdjson.RFC3339Time(now)
	if err := jsonv2.Unmarshal([]byte("null"), &unchanged); err != nil {
		t.Fatalf("null: %v", err)
	}
	if got := time.Time(unchanged); !got.Equal(now) {
		t.Errorf("null: got %v, want it left alone", got)
	}

	// Anything that is not a string is reported.
	for _, in := range []string{`12345`, `true`, `[]`, `{}`, `"not-a-time"`} {
		var back cmdjson.RFC3339Time
		if err := jsonv2.Unmarshal([]byte(in), &back); err == nil {
			t.Errorf("%v: expected an error, got %v", in, time.Time(back))
		}
	}

	// It satisfies the json/v2 interfaces.
	var _ jsonv2.MarshalerTo = cmdjson.RFC3339Time{}
	var _ jsonv2.UnmarshalerFrom = (*cmdjson.RFC3339Time)(nil)
}
