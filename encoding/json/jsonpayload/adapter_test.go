// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

type wrapped struct {
	A int    `json:"a"`
	B string `json:"b"`
}

// TestWrapperRoundTrip verifies that Wrapper delegates to the ordinary json
// encoding of the value it wraps, for both struct and non-struct types.
func TestWrapperRoundTrip(t *testing.T) {
	in := jsonpayload.Wrapper[wrapped]{Value: wrapped{A: 1, B: "two"}}
	buf, err := json.Marshal(&in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(buf), `{"a":1,"b":"two"}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	var out jsonpayload.Wrapper[wrapped]
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := out.Value, in.Value; got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestWrapperNonStructTypes(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		buf, err := json.Marshal(&jsonpayload.Wrapper[int]{Value: 42})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if got, want := string(buf), "42"; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
		var out jsonpayload.Wrapper[int]
		if err := json.Unmarshal(buf, &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if got, want := out.Value, 42; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("slice", func(t *testing.T) {
		buf, err := json.Marshal(&jsonpayload.Wrapper[[]string]{Value: []string{"a", "b"}})
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if got, want := string(buf), `["a","b"]`; got != want {
			t.Errorf("got %s, want %s", got, want)
		}
		var out jsonpayload.Wrapper[[]string]
		if err := json.Unmarshal(buf, &out); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if got, want := strings.Join(out.Value, ","), "a,b"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

// TestWrapperMarshalError verifies that an error from the underlying encoding
// is propagated rather than producing partial output.
func TestWrapperMarshalError(t *testing.T) {
	if _, err := json.Marshal(&jsonpayload.Wrapper[chan int]{Value: make(chan int)}); err == nil {
		t.Error("expected an error marshaling a channel, got nil")
	}
}

// TestWrapperUnmarshalError covers both failure modes: a value that cannot be
// read at all, and one that is well formed JSON but of the wrong type.
func TestWrapperUnmarshalError(t *testing.T) {
	t.Run("wrong type", func(t *testing.T) {
		var out jsonpayload.Wrapper[int]
		if err := json.Unmarshal([]byte(`"not-a-number"`), &out); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("truncated", func(t *testing.T) {
		var out jsonpayload.Wrapper[wrapped]
		if err := decodeString(&out, `{"a":1`); err == nil {
			t.Error("expected an error, got nil")
		}
	})
}
