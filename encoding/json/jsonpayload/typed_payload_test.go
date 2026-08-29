// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

// testPkg is the path of this external test package, which qualifies the
// names of all of the types declared in it.
const testPkg = "cloudeng.io/encoding/json/jsonpayload_test"

// payload is the primary well behaved message type used by these tests.
type payload struct {
	A int
}

func (p *payload) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(fmt.Sprintf(`{"A":%d}`, p.A)))
}

func (p *payload) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	// A distinct type is required to avoid recursing back into this method.
	type plain payload
	return json.Unmarshal(val, (*plain)(p))
}

// otherPayload is a second registered type, used to check that the type name
// in the envelope actually selects the type.
type otherPayload struct {
	B string
}

func (o *otherPayload) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(fmt.Sprintf(`{"B":%q}`, o.B)))
}

func (o *otherPayload) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain otherPayload
	return json.Unmarshal(val, (*plain)(o))
}

var errMarshalFailed = errors.New("marshal failed")

// failMarshal always fails to encode its payload.
type failMarshal struct{}

func (f *failMarshal) MarshalJSONTo(*jsontext.Encoder) error {
	return errMarshalFailed
}

var errUnmarshalFailed = errors.New("unmarshal failed")

// failUnmarshal encodes successfully but always fails to decode.
type failUnmarshal struct{}

func (f *failUnmarshal) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(`{}`))
}

func (f *failUnmarshal) UnmarshalJSONFrom(*jsontext.Decoder) error {
	return errUnmarshalFailed
}

// noUnmarshal is registered but cannot be decoded into, since its pointer does
// not implement json.UnmarshalerFrom.
type noUnmarshal struct {
	A int
}

func (n *noUnmarshal) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(fmt.Sprintf(`{"A":%d}`, n.A)))
}

// notRegistered is encodable and decodable but deliberately never registered.
type notRegistered struct {
	A int
}

func (n *notRegistered) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(fmt.Sprintf(`{"A":%d}`, n.A)))
}

func (n *notRegistered) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain notRegistered
	return json.Unmarshal(val, (*plain)(n))
}

func init() {
	jsonpayload.RegisterType[payload]()
	jsonpayload.RegisterType[otherPayload]()
	jsonpayload.RegisterType[failUnmarshal]()
	jsonpayload.RegisterType[noUnmarshal]()
	jsonpayload.RegisterType[jsonpayload.Wrapper[wrapped]]()
}

// decodeString feeds input to u through a streaming decoder. Unlike
// json.Unmarshal it does not require input to be a complete JSON value, so it
// can exercise the behaviour of a truncated message.
func decodeString(u json.UnmarshalerFrom, input string) error {
	return u.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader(input)))
}

// TestWriterEnvelope pins the wire format: a type name and a payload, in that
// order, in a single object.
func TestWriterEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  func() ([]byte, error)
		want string
	}{
		{
			"payload",
			func() ([]byte, error) { return json.Marshal(jsonpayload.NewWriter(&payload{A: 42})) },
			`{"type":"` + testPkg + `.payload","payload":{"A":42}}`,
		},
		{
			"other payload",
			func() ([]byte, error) {
				return json.Marshal(jsonpayload.NewWriter(&otherPayload{B: "hello"}))
			},
			`{"type":"` + testPkg + `.otherPayload","payload":{"B":"hello"}}`,
		},
		{
			// A wrapped type carries the full generic name, as documented on
			// Wrapper.
			"wrapper",
			func() ([]byte, error) {
				return json.Marshal(jsonpayload.NewWriter(
					&jsonpayload.Wrapper[wrapped]{Value: wrapped{A: 1, B: "two"}}))
			},
			`{"type":"cloudeng.io/encoding/json/jsonpayload.Wrapper[` + testPkg +
				`.wrapped]","payload":{"a":1,"b":"two"}}`,
		},
	} {
		buf, err := tc.got()
		if err != nil {
			t.Errorf("%v: Marshal: %v", tc.name, err)
			continue
		}
		if got, want := string(buf), tc.want; got != want {
			t.Errorf("%v:\n got %s\nwant %s", tc.name, got, want)
		}
	}
}

// TestWriterPayloadError verifies that a failure to encode the payload is
// returned to the caller rather than being masked.
func TestWriterPayloadError(t *testing.T) {
	_, err := json.Marshal(jsonpayload.NewWriter(&failMarshal{}))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, errMarshalFailed) {
		t.Errorf("got %v, want it to wrap %v", err, errMarshalFailed)
	}
}

// TestReaderRoundTrip covers the round trip that Writer and Reader are meant
// to be used as a pair for.
func TestReaderRoundTrip(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&payload{A: 42}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	rd := jsonpayload.NewReader[payload]()
	if err := json.Unmarshal(buf, &rd); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rd.Value == nil {
		t.Fatal("Value was not set")
	}
	if got, want := *rd.Value, (payload{A: 42}); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestReaderTypeMismatch verifies that a message for a different type is
// rejected rather than being coerced.
func TestReaderTypeMismatch(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&otherPayload{B: "hello"}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	rd := jsonpayload.NewReader[payload]()
	err = json.Unmarshal(buf, &rd)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "expected type name") {
		t.Errorf("got %v, want it to mention the expected type name", got)
	}
	if rd.Value != nil {
		t.Errorf("Value was set despite the mismatch: %+v", rd.Value)
	}
}

// TestReaderUnregisteredType verifies the error when a message names a type
// that was never registered, so there is nothing to construct.
func TestReaderUnregisteredType(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&notRegistered{A: 1}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	rd := jsonpayload.NewReader[notRegistered]()
	err = json.Unmarshal(buf, &rd)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "no registered type") {
		t.Errorf("got %v, want it to mention that no type is registered", got)
	}
}

// TestReaderPayloadError verifies that an error from decoding the payload
// itself is propagated.
func TestReaderPayloadError(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&failUnmarshal{}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	rd := jsonpayload.NewReader[failUnmarshal]()
	err = json.Unmarshal(buf, &rd)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, errUnmarshalFailed) {
		t.Errorf("got %v, want it to wrap %v", err, errUnmarshalFailed)
	}
}

// malformedEnvelopes are inputs that must be rejected before the payload is
// reached. An empty want matches any error, for the cases where the message is
// truncated and the exact error comes from the decoder.
var malformedEnvelopes = []struct {
	name, input, want string
}{
	{"empty", "", ""},
	{"not an object", `["a"]`, "expected '{'"},
	{"string", `"hello"`, "expected '{'"},
	{"truncated after brace", `{`, ""},
	{"empty object", `{}`, "expected 'type' key"},
	{"missing type key", `{"other":1}`, "expected 'type' key"},
	{"non-string type name", `{"type":123}`, "expected type name string"},
	{"truncated after type key", `{"type"`, ""},
	{"truncated after type name", `{"type":"x"`, ""},
	{"missing payload key", `{"type":"x","other":1}`, "expected 'payload' key"},
}

func TestReaderMalformed(t *testing.T) {
	for _, tc := range malformedEnvelopes {
		rd := jsonpayload.NewReader[payload]()
		err := decodeString(&rd, tc.input)
		if err == nil {
			t.Errorf("%v: expected an error, got nil", tc.name)
			continue
		}
		if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want it to contain %q", tc.name, err, tc.want)
		}
	}
}

// TestReaderTrailingContent verifies that the envelope must end after the
// payload.
func TestReaderTrailingContent(t *testing.T) {
	prefix := `{"type":"` + testPkg + `.payload","payload":{"A":1}`
	for _, tc := range []struct {
		name, input, want string
	}{
		{"missing end object", prefix, ""},
		{"extra member", prefix + `,"extra":1}`, "expected '}'"},
	} {
		rd := jsonpayload.NewReader[payload]()
		err := decodeString(&rd, tc.input)
		if err == nil {
			t.Errorf("%v: expected an error, got nil", tc.name)
			continue
		}
		if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want it to contain %q", tc.name, err, tc.want)
		}
	}
}

// TestReaderAnyRoundTrip verifies that ReaderAny selects the concrete type
// named by the message, without the caller knowing it in advance.
func TestReaderAnyRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		buf  func() ([]byte, error)
		want any
	}{
		{"payload", func() ([]byte, error) {
			return json.Marshal(jsonpayload.NewWriter(&payload{A: 42}))
		}, &payload{A: 42}},
		{"other payload", func() ([]byte, error) {
			return json.Marshal(jsonpayload.NewWriter(&otherPayload{B: "hello"}))
		}, &otherPayload{B: "hello"}},
		{"wrapper", func() ([]byte, error) {
			return json.Marshal(jsonpayload.NewWriter(
				&jsonpayload.Wrapper[wrapped]{Value: wrapped{A: 1, B: "two"}}))
		}, &jsonpayload.Wrapper[wrapped]{Value: wrapped{A: 1, B: "two"}}},
	} {
		buf, err := tc.buf()
		if err != nil {
			t.Errorf("%v: Marshal: %v", tc.name, err)
			continue
		}
		var rd jsonpayload.ReaderAny
		if err := json.Unmarshal(buf, &rd); err != nil {
			t.Errorf("%v: Unmarshal: %v", tc.name, err)
			continue
		}
		if got, want := fmt.Sprintf("%T%+v", rd.Value, rd.Value), fmt.Sprintf("%T%+v", tc.want, tc.want); got != want {
			t.Errorf("%v: got %v, want %v", tc.name, got, want)
		}
	}
}

// TestReaderAnyUnregisteredType verifies the error when the named type is not
// in the registry; ReaderAny has no compile time type to fall back on.
func TestReaderAnyUnregisteredType(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&notRegistered{A: 1}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var rd jsonpayload.ReaderAny
	err = json.Unmarshal(buf, &rd)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "no registered type") {
		t.Errorf("got %v, want it to mention that no type is registered", got)
	}
}

// TestReaderAnyNotUnmarshaler verifies that a registered type that cannot be
// decoded into is reported rather than panicking on the type assertion.
func TestReaderAnyNotUnmarshaler(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&noUnmarshal{A: 1}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var rd jsonpayload.ReaderAny
	err = json.Unmarshal(buf, &rd)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "not of expected type") {
		t.Errorf("got %v, want it to report the wrong type", got)
	}
}

func TestReaderAnyPayloadError(t *testing.T) {
	buf, err := json.Marshal(jsonpayload.NewWriter(&failUnmarshal{}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var rd jsonpayload.ReaderAny
	if err := json.Unmarshal(buf, &rd); !errors.Is(err, errUnmarshalFailed) {
		t.Errorf("got %v, want it to wrap %v", err, errUnmarshalFailed)
	}
}

func TestReaderAnyMalformed(t *testing.T) {
	for _, tc := range malformedEnvelopes {
		var rd jsonpayload.ReaderAny
		err := decodeString(&rd, tc.input)
		if err == nil {
			t.Errorf("%v: expected an error, got nil", tc.name)
			continue
		}
		if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: got %v, want it to contain %q", tc.name, err, tc.want)
		}
	}
}

// failAfterWriter fails once it has accepted n bytes, so that a failure can be
// injected at each stage of encoding the envelope.
type failAfterWriter struct {
	n int
}

var errWriteFailed = errors.New("write failed")

func (f *failAfterWriter) Write(p []byte) (int, error) {
	if len(p) > f.n {
		n := f.n
		f.n = 0
		return n, errWriteFailed
	}
	f.n -= len(p)
	return len(p), nil
}

// TestWriterEncoderError verifies that a failure of the underlying encoder is
// returned, wherever in the envelope it occurs.
func TestWriterEncoderError(t *testing.T) {
	// The envelope is a little over 70 bytes, so these limits interrupt it at
	// the opening brace, the type name, the payload and the closing brace.
	for _, limit := range []int{0, 1, 8, 20, 60, 70, 71} {
		w := jsonpayload.NewWriter(&payload{A: 42})
		err := w.MarshalJSONTo(jsontext.NewEncoder(&failAfterWriter{n: limit}))
		if err == nil {
			t.Errorf("limit %v: expected an error, got nil", limit)
			continue
		}
		if !errors.Is(err, errWriteFailed) {
			t.Errorf("limit %v: got %v, want it to wrap %v", limit, err, errWriteFailed)
		}
	}
}
