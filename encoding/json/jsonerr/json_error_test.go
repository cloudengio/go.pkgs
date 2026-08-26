// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonerr_test

import (
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonerr"
)

// testError is a concrete error type for use in tests.
type testError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *testError) Error() string { return e.Message }

func (e *testError) Is(target error) bool {
	_, ok := target.(*testError)
	return ok
}

// otherError is a second concrete type to verify type isolation.
type otherError struct {
	Reason string `json:"reason"`
}

func (e *otherError) Error() string { return e.Reason }

func (e *otherError) Is(target error) bool {
	_, ok := target.(*otherError)
	return ok
}

func init() {
	jsonerr.RegisterErrorType[testError, *testError]()
	jsonerr.RegisterErrorType[otherError, *otherError]()
}

func TestTypeNameForError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := jsonerr.TypeNameForError(nil); got != "" {
			t.Errorf("nil: got %q, want \"\"", got)
		}
	})

	t.Run("pointer receiver", func(t *testing.T) {
		got := jsonerr.TypeNameForError(&testError{})
		if !strings.HasSuffix(got, ".testError") {
			t.Errorf("got %q, want suffix \".testError\"", got)
		}
		if !strings.Contains(got, "cloudeng.io/encoding/json/jsonerr_test") {
			t.Errorf("got %q, want pkg path to contain \"cloudeng.io/encoding/json/jsonerr_test\"", got)
		}
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		var e *testError
		got := jsonerr.TypeNameForError(e)
		if !strings.HasSuffix(got, ".testError") {
			t.Errorf("typed nil: got %q, want suffix \".testError\"", got)
		}
	})
}

func TestMarshalError(t *testing.T) {
	orig := &testError{Code: 42, Message: "something broke"}
	data, err := jsonerr.MarshalError(orig)
	if err != nil {
		t.Fatalf("MarshalError: %v", err)
	}

	// Must be valid JSON.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	// Must have "error", "type", and "detail" fields.
	for _, key := range []string{"error", "type", "detail"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing field %q in marshaled output", key)
		}
	}

	// The "error" field must be the error message.
	var msg string
	if err := json.Unmarshal(raw["error"], &msg); err != nil || msg != orig.Message {
		t.Errorf("\"error\" field: got %q, want %q", msg, orig.Message)
	}
}

func TestRoundtrip(t *testing.T) {
	orig := &testError{Code: 7, Message: "roundtrip test"}

	data, err := jsonerr.MarshalError(orig)
	if err != nil {
		t.Fatalf("MarshalError: %v", err)
	}

	got, err := jsonerr.UnmarshalError(data)
	if err != nil {
		t.Fatalf("UnmarshalError: %v", err)
	}

	// errors.Is matches on type regardless of pointer identity.
	if !errors.Is(got, &testError{}) {
		t.Error("errors.Is(&testError{}) = false after roundtrip, want true")
	}

	// errors.As recovers the concrete type with fields intact.
	var te *testError
	if !errors.As(got, &te) {
		t.Fatalf("errors.As(*testError) = false after roundtrip, want true")
	}
	if te.Code != orig.Code {
		t.Errorf("Code: got %d, want %d", te.Code, orig.Code)
	}
	if te.Message != orig.Message {
		t.Errorf("Message: got %q, want %q", te.Message, orig.Message)
	}

	// Must not match a different registered type.
	if errors.Is(got, &otherError{}) {
		t.Error("errors.Is(&otherError{}) = true, want false")
	}
}

func TestUnmarshalErrorUnknownType(t *testing.T) {
	data := []byte(`{"error":"oops","type":"no.such.Type","detail":{}}`)
	got, err := jsonerr.UnmarshalError(data)
	if err != nil {
		t.Fatalf("UnmarshalError: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil error for unknown type, got nil")
	}
	if errors.Is(got, &testError{}) {
		t.Error("unknown type decoded as testError, want generic error")
	}
}

func TestMarshalJSONTo(t *testing.T) {
	orig := jsonerr.Error{
		Error:  "something broke",
		Type:   "some.pkg.MyError",
		Detail: jsontext.Value(`{"code":42}`),
	}

	var buf strings.Builder
	enc := jsontext.NewEncoder(&buf)
	if err := orig.MarshalJSONTo(enc); err != nil {
		t.Fatalf("MarshalJSONTo: %v", err)
	}

	dec := jsontext.NewDecoder(strings.NewReader(buf.String()))
	var got jsonerr.Error
	if err := got.UnmarshalJSONFrom(dec); err != nil {
		t.Fatalf("UnmarshalJSONFrom: %v", err)
	}

	if got.Error != orig.Error {
		t.Errorf("Error: got %q, want %q", got.Error, orig.Error)
	}
	if got.Type != orig.Type {
		t.Errorf("Type: got %q, want %q", got.Type, orig.Type)
	}

	var detail map[string]any
	if err := json.Unmarshal(got.Detail, &detail); err != nil {
		t.Fatalf("Detail is not valid JSON: %v (raw: %q)", err, got.Detail)
	}
	if got := detail["code"]; got != float64(42) {
		t.Errorf("Detail[\"code\"]: got %v, want 42", got)
	}
}

func TestMarshalJSONToEmptyDetail(t *testing.T) {
	orig := jsonerr.Error{Error: "oops", Type: "some.Type"}

	var buf strings.Builder
	enc := jsontext.NewEncoder(&buf)
	if err := orig.MarshalJSONTo(enc); err != nil {
		t.Fatalf("MarshalJSONTo: %v", err)
	}

	// detail should be null when Detail is empty
	if !strings.Contains(buf.String(), `"detail":null`) {
		t.Errorf("expected detail:null in output, got: %s", buf.String())
	}

	dec := jsontext.NewDecoder(strings.NewReader(buf.String()))
	var got jsonerr.Error
	if err := got.UnmarshalJSONFrom(dec); err != nil {
		t.Fatalf("UnmarshalJSONFrom: %v", err)
	}
	if got.Error != orig.Error {
		t.Errorf("Error: got %q, want %q", got.Error, orig.Error)
	}
}

func TestUnmarshalJSONFrom(t *testing.T) {
	input := `{"error":"msg","type":"some.Type","detail":{"code":3},"extra":"ignored"}`

	dec := jsontext.NewDecoder(strings.NewReader(input))
	var e jsonerr.Error
	if err := e.UnmarshalJSONFrom(dec); err != nil {
		t.Fatalf("UnmarshalJSONFrom: %v", err)
	}

	if e.Error != "msg" {
		t.Errorf("Error: got %q, want \"msg\"", e.Error)
	}
	if e.Type != "some.Type" {
		t.Errorf("Type: got %q, want \"some.Type\"", e.Type)
	}
	var detail map[string]any
	if err := json.Unmarshal(e.Detail, &detail); err != nil {
		t.Fatalf("Detail is not valid JSON: %v (raw: %q)", err, e.Detail)
	}
	if got := detail["code"]; got != float64(3) {
		t.Errorf("Detail[\"code\"]: got %v, want 3", got)
	}
}

func TestUnmarshalJSONFromUnknownFieldsSkipped(t *testing.T) {
	input := `{"unknown1":42,"error":"e","unknown2":{"nested":true},"type":"t","detail":null}`
	dec := jsontext.NewDecoder(strings.NewReader(input))
	var e jsonerr.Error
	if err := e.UnmarshalJSONFrom(dec); err != nil {
		t.Fatalf("UnmarshalJSONFrom with unknown fields: %v", err)
	}
	if e.Error != "e" {
		t.Errorf("Error: got %q, want \"e\"", e.Error)
	}
	if e.Type != "t" {
		t.Errorf("Type: got %q, want \"t\"", e.Type)
	}
}

func TestDefaultUnknownTypeHandler(t *testing.T) {
	// DefaultUnknownTypeHandler uses errors.New(err.Error), ignoring Type and Detail.
	env := jsonerr.Error{
		Error:  "the message",
		Type:   "some.Unknown.Type",
		Detail: jsontext.Value(`{"code":99}`),
	}
	got := jsonerr.DefaultUnknownTypeHandler(env)
	if got == nil {
		t.Fatal("got nil, want non-nil error")
	}
	if got.Error() != "the message" {
		t.Errorf("Error(): got %q, want %q", got.Error(), "the message")
	}
	// Must not be a registered concrete type.
	if errors.Is(got, &testError{}) {
		t.Error("result matched testError, want generic error")
	}
}

func TestNewUnmarshalErrorCustomHandler(t *testing.T) {
	// A custom handler can inspect Type and Detail to do something different.
	var received jsonerr.Error
	handler := func(env jsonerr.Error) error {
		received = env
		return &testError{Code: 999, Message: "handled: " + env.Error}
	}
	ue := jsonerr.NewUnmarshalError(handler)

	// Encode an unknown type.
	data := []byte(`{"error":"oops","type":"no.such.Package.NoType","detail":{"extra":true}}`)
	got, decErr := ue.Unmarshal(data)
	if decErr != nil {
		t.Fatalf("Unmarshal decode error: %v", decErr)
	}
	if got == nil {
		t.Fatal("got nil, want non-nil error from handler")
	}

	// The handler received the full envelope.
	if received.Type != "no.such.Package.NoType" {
		t.Errorf("received.Type = %q, want %q", received.Type, "no.such.Package.NoType")
	}

	// The returned error is whatever the handler produced.
	var te *testError
	if !errors.As(got, &te) {
		t.Fatalf("errors.As(*testError) = false, want true")
	}
	if te.Code != 999 {
		t.Errorf("Code: got %d, want 999", te.Code)
	}
	if te.Message != "handled: oops" {
		t.Errorf("Message: got %q, want %q", te.Message, "handled: oops")
	}
}

func TestUnmarshalErrorWithHandlerKnownType(t *testing.T) {
	// When the type IS registered, the custom handler is NOT invoked.
	handlerCalled := false
	handler := func(env jsonerr.Error) error {
		handlerCalled = true
		return errors.New("handler should not run")
	}
	ue := jsonerr.NewUnmarshalError(handler)

	orig := &testError{Code: 5, Message: "custom handler test"}
	data, err := jsonerr.MarshalError(orig)
	if err != nil {
		t.Fatalf("MarshalError: %v", err)
	}

	got, decErr := ue.Unmarshal(data)
	if decErr != nil {
		t.Fatalf("Unmarshal decode error: %v", decErr)
	}
	if handlerCalled {
		t.Error("handler was called for a known type, want it skipped")
	}

	var te *testError
	if !errors.As(got, &te) {
		t.Fatalf("errors.As(*testError) = false, want true")
	}
	if te.Code != orig.Code {
		t.Errorf("Code: got %d, want %d", te.Code, orig.Code)
	}
}

func TestNewUnmarshalErrorNilHandlerUsesDefault(t *testing.T) {
	// Passing nil handler falls back to DefaultUnknownTypeHandler.
	ue := jsonerr.NewUnmarshalError(nil)
	data := []byte(`{"error":"fallback message","type":"unknown.Type","detail":null}`)
	got, decErr := ue.Unmarshal(data)
	if decErr != nil {
		t.Fatalf("Unmarshal decode error: %v", decErr)
	}
	if got == nil {
		t.Fatal("got nil, want non-nil error")
	}
	if got.Error() != "fallback message" {
		t.Errorf("Error(): got %q, want \"fallback message\"", got.Error())
	}
}
