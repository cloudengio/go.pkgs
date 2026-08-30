// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonerr_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"encoding/json/v2"

	"cloudeng.io/encoding/json/jsonerr"
	"cloudeng.io/encoding/json/jsonpayload"
)

const testPkg = "cloudeng.io/encoding/json/jsonerr_test"

// notFound is an ordinary error type: a struct with an Error method and no
// JSON methods of its own.
type notFound struct {
	Name string `json:"name"`
	Code int    `json:"code"`
}

func (e *notFound) Error() string { return fmt.Sprintf("%v not found (%v)", e.Name, e.Code) }

func (e *notFound) Is(target error) bool {
	other, ok := target.(*notFound)
	return ok && (other.Name == "" || other.Name == e.Name)
}

// denied is a second registered type, to check that the type name selects the
// error that is reconstructed.
type denied struct {
	User string `json:"user"`
}

func (e *denied) Error() string { return "denied: " + e.User }

// unregistered is encodable but deliberately never registered.
type unregistered struct {
	Why string `json:"why"`
}

func (e *unregistered) Error() string { return "unregistered: " + e.Why }

// notAnError is registered but is not an error.
type notAnError struct {
	A int `json:"a"`
}

func init() {
	jsonpayload.RegisterType[notFound]()
	jsonpayload.RegisterType[denied]()
	jsonpayload.RegisterType[notAnError]()
}

// TestRoundTripRegistered verifies that a registered error is reconstructed as
// its original concrete type, so that errors.Is and errors.As work on it.
func TestRoundTripRegistered(t *testing.T) {
	orig := &notFound{Name: "widget", Code: 404}
	buf, err := jsonerr.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"error":"widget not found (404)","detail":{"type":"` + testPkg +
		`.notFound","payload":{"name":"widget","code":404}}}`
	if got := string(buf); got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}

	got, err := jsonerr.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var target *notFound
	if !errors.As(got, &target) {
		t.Fatalf("errors.As: got %T, want *notFound", got)
	}
	if *target != *orig {
		t.Errorf("got %+v, want %+v", *target, *orig)
	}
	if !errors.Is(got, &notFound{}) {
		t.Error("errors.Is failed for the decoded error")
	}
	if got, want := got.Error(), orig.Error(); got != want {
		t.Errorf("message: got %q, want %q", got, want)
	}
}

// TestRoundTripSelectsType verifies that the type name in the message chooses
// which error is reconstructed.
func TestRoundTripSelectsType(t *testing.T) {
	for _, orig := range []error{
		&notFound{Name: "a", Code: 1},
		&denied{User: "bob"},
	} {
		buf, err := jsonerr.Marshal(orig)
		if err != nil {
			t.Errorf("%T: Marshal: %v", orig, err)
			continue
		}
		got, err := jsonerr.Unmarshal(buf)
		if err != nil {
			t.Errorf("%T: Unmarshal: %v", orig, err)
			continue
		}
		if fmt.Sprintf("%T", got) != fmt.Sprintf("%T", orig) {
			t.Errorf("got %T, want %T", got, orig)
		}
		if got.Error() != orig.Error() {
			t.Errorf("message: got %q, want %q", got.Error(), orig.Error())
		}
	}
}

// TestUnregisteredTypeDegradesToMessage verifies that an error whose type is
// not registered here is still usable: the payload is carried but ignored and
// the message survives. A receiver that registers the type later can decode
// the same bytes in full.
func TestUnregisteredTypeDegradesToMessage(t *testing.T) {
	orig := &unregistered{Why: "nope"}
	buf, err := jsonerr.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	// The payload is still written, so the message is forward compatible.
	if got, want := string(buf), testPkg+".unregistered"; !strings.Contains(got, want) {
		t.Errorf("got %s, want it to carry %s", got, want)
	}

	got, err := jsonerr.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Error() != orig.Error() {
		t.Errorf("message: got %q, want %q", got.Error(), orig.Error())
	}
	var target *unregistered
	if errors.As(got, &target) {
		t.Error("the concrete type should not be recovered for an unregistered type")
	}
}

// TestErrorsWithoutEncodableState verifies that an error with no exported
// state, which cannot be marshaled, is represented by its message rather than
// being reported as a failure.
func TestErrorsWithoutEncodableState(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"errors.New", errors.New("plain"), `{"error":"plain"}`},
		{"fmt.Errorf", fmt.Errorf("bad: %v", 1), `{"error":"bad: 1"}`},
		// Wrapping hides the wrapped error's state, so only the combined
		// message survives.
		{"wrapped", fmt.Errorf("ctx: %w", &notFound{Name: "w", Code: 1}),
			`{"error":"ctx: w not found (1)"}`},
	} {
		buf, err := jsonerr.Marshal(tc.err)
		if err != nil {
			t.Errorf("%v: Marshal: %v", tc.name, err)
			continue
		}
		if got := string(buf); got != tc.want {
			t.Errorf("%v: got %s, want %s", tc.name, got, tc.want)
		}
		got, err := jsonerr.Unmarshal(buf)
		if err != nil {
			t.Errorf("%v: Unmarshal: %v", tc.name, err)
			continue
		}
		if got.Error() != tc.err.Error() {
			t.Errorf("%v: message: got %q, want %q", tc.name, got.Error(), tc.err.Error())
		}
	}
}

// TestNil verifies that a nil error round trips as nil.
func TestNil(t *testing.T) {
	buf, err := jsonerr.Marshal(nil)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(buf), `{"error":""}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	got, err := jsonerr.Unmarshal(buf)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

// TestUnmarshalErrors covers the failures that are reported rather than
// degraded.
func TestUnmarshalErrors(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		if _, err := jsonerr.Unmarshal([]byte(`{`)); err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("payload does not match its type", func(t *testing.T) {
		buf := []byte(`{"error":"x","detail":{"type":"` + testPkg +
			`.notFound","payload":{"name":[1,2]}}}`)
		if _, err := jsonerr.Unmarshal(buf); err == nil {
			t.Error("expected an error, got nil")
		} else if !strings.Contains(err.Error(), "decode") {
			t.Errorf("got %v, want it to mention decoding", err)
		}
	})

	t.Run("registered type is not an error", func(t *testing.T) {
		buf := []byte(`{"error":"x","detail":{"type":"` + testPkg +
			`.notAnError","payload":{"a":1}}}`)
		if _, err := jsonerr.Unmarshal(buf); err == nil {
			t.Error("expected an error, got nil")
		} else if !strings.Contains(err.Error(), "is not an error") {
			t.Errorf("got %v, want it to report a non-error type", err)
		}
	})
}

// TestWireIsTheRepresentation verifies that Wire is what is actually produced
// and accepted, so that another package can build or inspect these messages.
func TestWireIsTheRepresentation(t *testing.T) {
	orig := &notFound{Name: "widget", Code: 404}
	buf, err := jsonerr.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// What is produced is the marshaling of the Wire for that error.
	fromWire, err := json.Marshal(jsonerr.WireForError(orig))
	if err != nil {
		t.Fatalf("Marshal(Wire): %v", err)
	}
	if got, want := string(buf), string(fromWire); got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}

	// And a Wire built by hand is accepted.
	hand := jsonerr.Wire{
		Message: "widget not found (404)",
		Detail: &jsonpayload.Wire{
			Type:    testPkg + ".notFound",
			Payload: []byte(`{"name":"widget","code":404}`),
		},
	}
	decoded, err := jsonerr.ErrorForWire(hand)
	if err != nil {
		t.Fatalf("ErrorForWire: %v", err)
	}
	var target *notFound
	if !errors.As(decoded, &target) {
		t.Fatalf("got %T, want *notFound", decoded)
	}
	if *target != *orig {
		t.Errorf("got %+v, want %+v", *target, *orig)
	}
}

// response carries an error as an ordinary tagged field.
type response struct {
	Result string             `json:"result"`
	Err    jsonerr.ReadWriter `json:"err"`
}

// TestReadWriterField verifies that an error can be a field of a struct that
// is itself encoded and decoded as JSON.
func TestReadWriterField(t *testing.T) {
	in := response{Result: "partial", Err: jsonerr.ReadWriter{Err: &denied{User: "bob"}}}
	buf, err := json.Marshal(&in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"result":"partial","err":{"error":"denied: bob","detail":{"type":"` +
		testPkg + `.denied","payload":{"user":"bob"}}}}`
	if got := string(buf); got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}

	var out response
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := out.Result, in.Result; got != want {
		t.Errorf("Result: got %q, want %q", got, want)
	}
	var target *denied
	if !errors.As(out.Err.Err, &target) {
		t.Fatalf("got %T, want *denied", out.Err.Err)
	}
	if got, want := target.User, "bob"; got != want {
		t.Errorf("User: got %q, want %q", got, want)
	}
}

// TestReadWriterFieldNilError verifies that a struct with no error encodes and
// decodes as one.
func TestReadWriterFieldNilError(t *testing.T) {
	buf, err := json.Marshal(&response{Result: "ok"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(buf), `{"result":"ok","err":{"error":""}}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	var out response
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Err.Err != nil {
		t.Errorf("got %v, want a nil error", out.Err.Err)
	}
}
