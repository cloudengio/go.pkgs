// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

// typedEnvelope carries a message whose type is known at compile time, as an
// ordinary tagged field.
type typedEnvelope struct {
	ID      int                                       `json:"id"`
	Message jsonpayload.ReadWriter[payload, *payload] `json:"message"`
	Trailer string                                    `json:"trailer"`
}

// anyEnvelope carries a message whose type is not known until it is read.
type anyEnvelope struct {
	ID      int                       `json:"id"`
	Message jsonpayload.ReadWriterAny `json:"message"`
	Trailer string                    `json:"trailer"`
}

const wantTypedEnvelope = `{"id":7,"message":{"type":"` + testPkg +
	`.payload","payload":{"A":42}},"trailer":"end"}`

// TestReadWriterRoundTrip verifies that a ReadWriter field encodes and decodes
// as an ordinary field of the enclosing struct, and that the members either
// side of it are unaffected.
func TestReadWriterRoundTrip(t *testing.T) {
	in := typedEnvelope{
		ID:      7,
		Message: jsonpayload.NewReadWriter[payload, *payload](payload{A: 42}),
		Trailer: "end",
	}
	buf, err := json.Marshal(&in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(buf), wantTypedEnvelope; got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}

	// A zero valued envelope decodes without anything being set up first:
	// the message is decoded into the field in place.
	var out typedEnvelope
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := out, in; got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got, want := out.Message.Value.A, 42; got != want {
		t.Errorf("payload: got %v, want %v", got, want)
	}
}

// TestReadWriterTypeMismatch verifies that a message for a different type is
// reported rather than decoded into the field.
func TestReadWriterTypeMismatch(t *testing.T) {
	buf := []byte(`{"id":1,"message":{"type":"` + testPkg +
		`.otherPayload","payload":{"B":"x"}},"trailer":"end"}`)
	var out typedEnvelope
	err := json.Unmarshal(buf, &out)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "expected type name") {
		t.Errorf("got %v, want it to mention the expected type name", got)
	}
}

// TestReadWriterAnyRoundTrip verifies the same for a message whose type is
// resolved through the registry when it is read.
func TestReadWriterAnyRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name     string
		val      jsonpayload.ReaderWriter
		wantType string
		want     string
	}{
		{"payload", &payload{A: 42}, testPkg + ".payload", `{"id":7,"message":{"type":"` + testPkg +
			`.payload","payload":{"A":42}},"trailer":"end"}`},
		{"other payload", &otherPayload{B: "hi"}, testPkg + ".otherPayload", `{"id":7,"message":{"type":"` + testPkg +
			`.otherPayload","payload":{"B":"hi"}},"trailer":"end"}`},
	} {
		in := anyEnvelope{ID: 7, Message: jsonpayload.NewReadWriterAny(tc.val), Trailer: "end"}
		buf, err := json.Marshal(&in)
		if err != nil {
			t.Errorf("%v: Marshal: %v", tc.name, err)
			continue
		}
		if got, want := string(buf), tc.want; got != want {
			t.Errorf("%v:\n got %s\nwant %s", tc.name, got, want)
		}

		var out anyEnvelope
		if err := json.Unmarshal(buf, &out); err != nil {
			t.Errorf("%v: Unmarshal: %v", tc.name, err)
			continue
		}
		if got, want := out.ID, in.ID; got != want {
			t.Errorf("%v: ID: got %v, want %v", tc.name, got, want)
		}
		if got, want := out.Trailer, in.Trailer; got != want {
			t.Errorf("%v: Trailer: got %v, want %v", tc.name, got, want)
		}
		if got, want := out.Message.Type, tc.wantType; got != want {
			t.Errorf("%v: Type: got %q, want %q", tc.name, got, want)
		}
		// The concrete type is recovered from the message.
		if got, want := typeOf(out.Message.Value), typeOf(tc.val); got != want {
			t.Errorf("%v: got %v, want %v", tc.name, got, want)
		}
	}
}

// typeOf reports the dynamic type of v without using this package's own
// naming, so the round trip is checked independently of it.
func typeOf(v any) string { return fmt.Sprintf("%T", v) }

// TestReadWriterAnyErrors covers the failures specific to the dynamic variant.
func TestReadWriterAnyErrors(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		in := anyEnvelope{ID: 1, Trailer: "end"}
		if _, err := json.Marshal(&in); err == nil {
			t.Error("expected an error marshaling a nil message, got nil")
		} else if !strings.Contains(err.Error(), "is nil") {
			t.Errorf("got %v, want it to report a nil value", err)
		}
	})

	t.Run("unregistered type", func(t *testing.T) {
		buf := []byte(`{"id":1,"message":{"type":"example.com/pkg.Unknown","payload":{}},"trailer":"e"}`)
		var out anyEnvelope
		err := json.Unmarshal(buf, &out)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if got := err.Error(); !strings.Contains(got, "no registered type") {
			t.Errorf("got %v, want it to report an unregistered type", got)
		}
	})

	t.Run("registered but not encodable", func(t *testing.T) {
		// noUnmarshal is registered but implements only MarshalJSONTo, so it
		// cannot satisfy ReaderWriter.
		buf := []byte(`{"id":1,"message":{"type":"` + testPkg +
			`.noUnmarshal","payload":{"A":1}},"trailer":"e"}`)
		var out anyEnvelope
		if err := json.Unmarshal(buf, &out); err == nil {
			t.Error("expected an error, got nil")
		}
	})
}
