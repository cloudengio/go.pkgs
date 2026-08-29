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
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("type")); err != nil {
		return err
	}
	tn := types.TypeName[T]()
	if err := enc.WriteToken(jsontext.String(tn)); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("payload")); err != nil {
		return err
	}
	if err := w.Value.MarshalJSONTo(enc); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// Reader is a JSON decoder for typed messages. It should be used when the
// expected type is known at compile time. T is the type decoded into, which
// is typically a pointer type since UnmarshalJSONFrom is usually implemented
// on a pointer receiver; the type name expected in the message is that of the
// value type either way, since TypeName removes pointer indirection.
type Reader[T json.UnmarshalerFrom] struct {
	Value T
}

// NewReader returns a Reader for messages carrying a T.
func NewReader[T json.UnmarshalerFrom]() Reader[T] {
	return Reader[T]{}
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

func (r *Reader[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	typeName, err := readToPayload(dec)
	if err != nil {
		return err
	}
	tn := types.TypeName[T]()
	if typeName != tn {
		return fmt.Errorf("expected type name %q, got %q", tn, typeName)
	}
	if val, ok := New(tn); !ok {
		return fmt.Errorf("no registered type for %q", tn)
	} else if r.Value, ok = val.(T); !ok {
		return fmt.Errorf("registered type for %q is not of expected type", tn)
	}
	if err := r.Value.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	return readToEndObject(dec)
}

// ReaderAny is a JSON decoder for typed messages. It should be used when the
// expected type is not known at compile time. The Value field will be set to
// the decoded value.
type ReaderAny struct {
	Value json.UnmarshalerFrom
}

func (r *ReaderAny) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	typeName, err := readToPayload(dec)
	if err != nil {
		return err
	}
	if val, ok := New(typeName); !ok {
		return fmt.Errorf("no registered type for %q", typeName)
	} else if r.Value, ok = val.(json.UnmarshalerFrom); !ok {
		return fmt.Errorf("registered type for %q is not of expected type", typeName)
	}
	if err := r.Value.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	return readToEndObject(dec)
}

/*

	expectedType := extprotocol.TypeName[RT]()
		if resp.Type != expectedType {
			return nil, fmt.Errorf("extpv0: response type mismatch: expected %s, got %s", expectedType, resp.Type)
		}
		result := RPT(new(RT))
		dec := jsontext.NewDecoder(bytes.NewBuffer(resp.Payload))
		if err := result.UnmarshalJSONFrom(dec); err != nil {
			return nil, fmt.Errorf("extpv0: unmarshal response: %w", err)
		}
		return (*RT)(result), resp.Error
*/
