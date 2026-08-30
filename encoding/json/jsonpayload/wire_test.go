// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

// wireFor returns the Wire value that this package documents as the encoding
// of val, expressed independently of the token writing that produces it.
func wireFor(typeName, payload string) jsonpayload.Wire {
	return jsonpayload.Wire{
		Type:    typeName,
		Payload: jsontext.Value(payload),
	}
}

// TestWireIsWhatIsEncoded verifies that the bytes the writers produce are
// exactly those of the equivalent Wire value, so that Wire really is the
// representation this package emits, including its field names and their
// order. The writers construct the message token by token rather than by
// marshaling a Wire, so nothing but this test keeps the two in step.
func TestWireIsWhatIsEncoded(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  func() ([]byte, error)
		want jsonpayload.Wire
	}{
		{
			"Writer",
			func() ([]byte, error) { return json.Marshal(jsonpayload.NewWriter(&payload{A: 42})) },
			wireFor(testPkg+".payload", `{"A":42}`),
		},
		{
			"WriterAny",
			func() ([]byte, error) { return json.Marshal(jsonpayload.NewWriterAny(&payload{A: 42})) },
			wireFor(testPkg+".payload", `{"A":42}`),
		},
		{
			"Writer with a wrapped value",
			func() ([]byte, error) {
				return json.Marshal(jsonpayload.NewWriter(
					&jsonpayload.Wrapper[wrapped]{Value: wrapped{A: 1, B: "two"}}))
			},
			wireFor("cloudeng.io/encoding/json/jsonpayload.Wrapper["+testPkg+".wrapped]",
				`{"a":1,"b":"two"}`),
		},
	} {
		got, err := tc.got()
		if err != nil {
			t.Errorf("%v: Marshal: %v", tc.name, err)
			continue
		}
		want, err := json.Marshal(tc.want)
		if err != nil {
			t.Errorf("%v: Marshal(Wire): %v", tc.name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%v:\n got %s\nwant %s", tc.name, got, want)
		}
	}
}

// TestWireDecodesWhatIsEncoded verifies the same thing from the other side:
// what the writers emit unmarshals into a Wire with the expected fields.
func TestWireDecodesWhatIsEncoded(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&payload{A: 42}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var w jsonpayload.Wire
	if err := json.Unmarshal(buf, &w); err != nil {
		t.Fatalf("Unmarshal into Wire: %v", err)
	}
	if got, want := w.Type, testPkg+".payload"; got != want {
		t.Errorf("Type: got %q, want %q", got, want)
	}
	if got, want := string(w.Payload), `{"A":42}`; got != want {
		t.Errorf("Payload: got %s, want %s", got, want)
	}
}

// TestWireIsWhatIsDecoded verifies that a message built from a Wire value,
// rather than by this package's writers, is accepted by every reader. Together
// with the tests above this pins both directions to the same representation.
func TestWireIsWhatIsDecoded(t *testing.T) {
	buf, err := json.Marshal(wireFor(testPkg+".payload", `{"A":7}`))
	if err != nil {
		t.Fatalf("Marshal(Wire): %v", err)
	}
	want := payload{A: 7}

	t.Run("Decode", func(t *testing.T) {
		var got payload
		if err := jsonpayload.Decode(jsontext.NewDecoder(bytes.NewReader(buf)), &got); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("Reader", func(t *testing.T) {
		var got payload
		rd := jsonpayload.NewReader(&got)
		if err := json.Unmarshal(buf, &rd); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("ReaderAny", func(t *testing.T) {
		var rd jsonpayload.ReaderAny
		if err := json.Unmarshal(buf, &rd); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		got, ok := rd.Value.(*payload)
		if !ok {
			t.Fatalf("got %T, want *payload", rd.Value)
		}
		if *got != want {
			t.Errorf("got %+v, want %+v", *got, want)
		}
	})
}

// TestWireFieldNames verifies the names Wire declares are the names on the
// wire, independently of Wire's own struct tags being correct.
func TestWireFieldNames(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&payload{A: 42}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]jsontext.Value
	if err := json.Unmarshal(buf, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := len(raw), 2; got != want {
		t.Errorf("member count: got %v, want %v (%s)", got, want, buf)
	}
	for _, name := range []string{"type", "payload"} {
		if _, ok := raw[name]; !ok {
			t.Errorf("missing member %q in %s", name, buf)
		}
	}
}
