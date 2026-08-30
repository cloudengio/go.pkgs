// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"

	"cloudeng.io/types"
)

// Wire represents the 'on-the-wire' representation of a typed message and
// documents the format that this package produces and accepts. Another
// package can marshal a Wire value to generate a message that the readers
// here will understand, or unmarshal a message into one to inspect it,
// without depending on the writers in this package.
//
// The readers and writers here work directly with the token stream rather
// than with a Wire value, so as not to buffer the payload; the tests verify
// that what they produce and accept is exactly this representation.
type Wire struct {
	Type    string         `json:"type"`
	Payload jsontext.Value `json:"payload"`
}

// Writer is a JSON encoder for typed messages. T is the type of the value
// being encoded, which is typically a pointer type since MarshalJSONTo is
// usually implemented on a pointer receiver; the type name written to the
// message is that of the value type either way, since TypeName removes
// pointer indirection.
type Writer[T json.MarshalerTo] struct {
	Value T
}

// NewWriter returns a Writer for val. The type argument is inferred from val.
func NewWriter[T json.MarshalerTo](val T) Writer[T] {
	return Writer[T]{Value: val}
}

func (w Writer[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return writeMessage(enc, types.TypeName[T](), w.Value)
}

// WriterAny is a JSON encoder for typed messages whose type is not known at
// compile time, such as messages held in a slice of some interface type. The
// name written is that of the value's dynamic type, whereas Writer uses the
// name of its type parameter, which for an interface typed variable would
// name the interface rather than the message it holds. Value must be non-nil.
type WriterAny struct {
	Value json.MarshalerTo
}

// NewWriterAny returns a WriterAny for val.
func NewWriterAny(val json.MarshalerTo) WriterAny {
	return WriterAny{Value: val}
}

func (w WriterAny) MarshalJSONTo(enc *jsontext.Encoder) error {
	tn := types.TypeNameForValue(w.Value)
	if tn == "" {
		return fmt.Errorf("no value to write: the value is nil")
	}
	return writeMessage(enc, tn, w.Value)
}

// writeMessage writes val as a typed message: an object carrying typeName
// alongside the value's own encoding.
func writeMessage(enc *jsontext.Encoder, typeName string, val json.MarshalerTo) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("type")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(typeName)); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("payload")); err != nil {
		return err
	}
	if err := val.MarshalJSONTo(enc); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// Unmarshaler is the constraint for a value that a typed message can be
// decoded into. It requires comparable so that a missing value can be
// detected by comparison with the zero value rather than by reflection; a
// decode target is in practice a pointer, which is always comparable.
type Unmarshaler interface {
	comparable
	json.UnmarshalerFrom
}

// Reader is a JSON decoder for typed messages that adapts Decode to the
// json.UnmarshalerFrom interface, for when a typed message appears within a
// larger JSON document, or is decoded with json.Unmarshal. To decode a
// message on its own, call Decode directly and avoid the wrapper entirely.
//
// Value is supplied by the caller and decoded into in place, so that decoding
// allocates nothing; it must be non-nil. Decoding twice into the same Reader
// decodes into the same value both times.
type Reader[T Unmarshaler] struct {
	Value T
}

// NewReader returns a Reader that decodes into val. The type argument is
// inferred from val.
func NewReader[T Unmarshaler](val T) Reader[T] {
	return Reader[T]{Value: val}
}

func readToPayload(dec *jsontext.Decoder) (string, error) {
	if tok, err := dec.ReadToken(); err != nil {
		return "", err
	} else if tok.Kind() != jsontext.KindBeginObject {
		return "", fmt.Errorf("expected '{', got %v", tok.Kind())
	}
	tok, err := dec.ReadToken()
	if err != nil {
		return "", err
	}
	if tok.Kind() != jsontext.KindString || tok.String() != "type" {
		return "", fmt.Errorf("expected 'type' key, got %v", tok)
	}
	tok, err = dec.ReadToken()
	if err != nil {
		return "", err
	}
	if tok.Kind() != jsontext.KindString {
		return "", fmt.Errorf("expected type name string, got %v", tok)
	}
	typeName := tok.String()
	tok, err = dec.ReadToken()
	if err != nil {
		return "", err
	}
	if tok.Kind() != jsontext.KindString || tok.String() != "payload" {
		return "", fmt.Errorf("expected 'payload' key, got %v", tok)
	}
	return typeName, nil
}

func readToEndObject(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	if tok.Kind() != jsontext.KindEndObject {
		return fmt.Errorf("expected '}', got %v", tok)
	}
	return nil
}

// Decode reads a typed message from dec into val, which must be non-nil. The
// message's type name must be that of T, and the payload is decoded into val
// in place, so that decoding allocates nothing. The type argument is inferred
// from val, and no registration is required since the expected type is known
// at compile time.
func Decode[T Unmarshaler](dec *jsontext.Decoder, val T) error {
	typeName, err := readToPayload(dec)
	if err != nil {
		return err
	}
	tn := types.TypeName[T]()
	if typeName != tn {
		return fmt.Errorf("expected type name %q, got %q", tn, typeName)
	}
	// Decoding is into the caller's value, so report a missing one rather
	// than leaving it to the payload's own decoder to fail, or panic, on a
	// nil receiver.
	var zero T
	if val == zero {
		return fmt.Errorf("no value to decode %q into: the value is nil", tn)
	}
	if err := val.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	return readToEndObject(dec)
}

func (r Reader[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return Decode(dec, r.Value)
}

// ReaderAny is a JSON decoder for typed messages. It should be used when the
// expected type is not known at compile time. The Value field will be set to
// the decoded value. All types that may be decoded must be registered with
// RegisterType.
type ReaderAny struct {
	Value json.UnmarshalerFrom
}

func (r *ReaderAny) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	typeName, err := readToPayload(dec)
	if err != nil {
		return err
	}
	if val, ok := NewInstance(typeName); !ok {
		return fmt.Errorf("no registered type for %q", typeName)
	} else if r.Value, ok = val.(json.UnmarshalerFrom); !ok {
		return fmt.Errorf("registered type for %q is not of expected type", typeName)
	}
	if err := r.Value.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	return readToEndObject(dec)
}

// ReaderWriter is a value that can both encode and decode itself as the
// payload of a typed message.
type ReaderWriter interface {
	json.MarshalerTo
	json.UnmarshalerFrom
}

// PointerReaderWriter constrains PT to be a pointer to T that implements
// ReaderWriter, which is how ReadWriter both encodes and decodes the value it
// holds without allocating one.
type PointerReaderWriter[T any] interface {
	comparable
	*T
	ReaderWriter
}

// ReadWriter is a typed message that can be both encoded and decoded, for use
// as a field of a struct that is itself marshaled as JSON:
//
//	type Envelope struct {
//		ID      int                                     `json:"id"`
//		Message jsonpayload.ReadWriter[Greeting, *Greeting] `json:"message"`
//	}
//
// Unlike Reader, the value is held by the field rather than pointed to by it,
// so a zero valued Envelope can be unmarshaled into directly: the payload is
// decoded in place and nothing is allocated. Both type arguments must be
// given, since Go infers type arguments for calls but not for types.
type ReadWriter[T any, PT PointerReaderWriter[T]] struct {
	Value T
}

// NewReadWriter returns a ReadWriter holding val, for use when encoding.
func NewReadWriter[T any, PT PointerReaderWriter[T]](val T) ReadWriter[T, PT] {
	return ReadWriter[T, PT]{Value: val}
}

func (rw ReadWriter[T, PT]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return writeMessage(enc, types.TypeName[T](), PT(&rw.Value))
}

func (rw *ReadWriter[T, PT]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	return Decode(dec, PT(&rw.Value))
}

// ReadWriterAny is a typed message that can be both encoded and decoded when
// its type is not known at compile time, for use as a field of a struct that
// is itself marshaled as JSON. Every type that may be decoded must be
// registered with RegisterType; Value must be non-nil to encode.
type ReadWriterAny struct {
	Value ReaderWriter
}

// NewReadWriterAny returns a ReadWriterAny holding val, for use when encoding.
func NewReadWriterAny(val ReaderWriter) ReadWriterAny {
	return ReadWriterAny{Value: val}
}

func (rw ReadWriterAny) MarshalJSONTo(enc *jsontext.Encoder) error {
	tn := types.TypeNameForValue(rw.Value)
	if tn == "" {
		return fmt.Errorf("no value to write: the value is nil")
	}
	return writeMessage(enc, tn, rw.Value)
}

func (rw *ReadWriterAny) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var rd ReaderAny
	if err := rd.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	val, ok := rd.Value.(ReaderWriter)
	if !ok {
		return fmt.Errorf("registered type for %q cannot be encoded",
			types.TypeNameForValue(rd.Value))
	}
	rw.Value = val
	return nil
}
