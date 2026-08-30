// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonerr provides support for sending errors over the wire as JSON.
//
// An encoded error carries two things: a message, which any receiver can use,
// and, when the error has state that can be encoded, a typed payload that a
// receiver which has registered the error's type can decode back into the
// original concrete error. The type of an error is identified by its fully
// qualified name (e.g. "example.com/pkg.MyError") and the payload uses the
// representation defined by cloudeng.io/encoding/json/jsonpayload.
//
// An error type is an ordinary struct with an Error method; it needs no JSON
// methods of its own, since its payload is encoded and decoded by the
// standard struct encoding. Types are registered for decoding with
// jsonpayload.RegisterType:
//
//	type NotFound struct {
//		Name string `json:"name"`
//	}
//
//	func (e *NotFound) Error() string { return e.Name + " not found" }
//
//	func init() { jsonpayload.RegisterType[NotFound]() }
package jsonerr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"

	"cloudeng.io/encoding/json/jsonpayload"
	"cloudeng.io/types"
)

// Wire is the 'on-the-wire' representation of an error and documents the
// format that this package produces and accepts. Message is always present so
// that a receiver can report something useful for an error whose type it does
// not know. Detail is present only when the error's state could be encoded,
// and is the representation used by jsonpayload.
type Wire struct {
	Message string            `json:"error"`
	Detail  *jsonpayload.Wire `json:"detail,omitempty"`
}

// WireForError returns the Wire representation of err. Encoding the error's
// state is best effort: an error such as one returned by errors.New or
// fmt.Errorf has no exported state to encode, and is represented by its
// message alone rather than being reported as a failure.
func WireForError(err error) Wire {
	if err == nil {
		return Wire{}
	}
	w := Wire{Message: err.Error()}
	tn := types.TypeNameForValue(err)
	if tn == "" {
		return w
	}
	payload, mErr := json.Marshal(err)
	if mErr != nil {
		return w
	}
	w.Detail = &jsonpayload.Wire{Type: tn, Payload: payload}
	return w
}

// ErrorForWire returns the error represented by w. If w carries a payload
// whose type has been registered with jsonpayload.RegisterType then the
// original concrete error is returned, so that errors.Is and errors.As can be
// used on it. Otherwise an error carrying only the message is returned, which
// means that an unregistered type degrades to its message rather than to a
// failure. A nil error is returned for the zero Wire.
func ErrorForWire(w Wire) (error, error) {
	if w.Message == "" && w.Detail == nil {
		return nil, nil
	}
	if w.Detail == nil {
		return errors.New(w.Message), nil
	}
	val, ok := jsonpayload.NewInstance(w.Detail.Type)
	if !ok {
		// The type is not registered here, so the payload cannot be decoded;
		// the message is still usable.
		return errors.New(w.Message), nil
	}
	if err := json.Unmarshal(w.Detail.Payload, val); err != nil {
		return nil, fmt.Errorf("decode %v: %w", w.Detail.Type, err)
	}
	decoded, ok := val.(error)
	if !ok {
		return nil, fmt.Errorf("registered type %v is not an error", w.Detail.Type)
	}
	return decoded, nil
}

// Marshal encodes err as its Wire representation.
func Marshal(err error) ([]byte, error) {
	return json.Marshal(WireForError(err))
}

// Unmarshal decodes an error encoded by Marshal. The outer error reports a
// failure to decode; the inner one is the error that was encoded.
func Unmarshal(data []byte) (error, error) {
	var w Wire
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return ErrorForWire(w)
}

// Writer encodes an error, for use where an error is a field of a struct that
// is itself encoded as JSON, or is otherwise written to an encoder.
type Writer struct {
	Err error
}

func (w Writer) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, WireForError(w.Err))
}

// Reader decodes an error encoded by Writer, for use where an error is a
// field of a struct that is itself decoded from JSON.
type Reader struct {
	Err error
}

func (r *Reader) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var wire Wire
	if err := json.UnmarshalDecode(dec, &wire); err != nil {
		return err
	}
	decoded, err := ErrorForWire(wire)
	if err != nil {
		return err
	}
	r.Err = decoded
	return nil
}

// ReadWriter is an error that can be both encoded and decoded, for use as an
// ordinary tagged field of a struct:
//
//	type Response struct {
//		Result string          `json:"result"`
//		Err    jsonerr.ReadWriter `json:"err"`
//	}
type ReadWriter struct {
	Err error
}

func (rw ReadWriter) MarshalJSONTo(enc *jsontext.Encoder) error {
	return Writer(rw).MarshalJSONTo(enc)
}

func (rw *ReadWriter) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var r Reader
	if err := r.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	rw.Err = r.Err
	return nil
}
