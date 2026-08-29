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

// Marshaler is a type constraint for encoding typed messages.
type Marshaler[T any] interface {
	*T
	json.MarshalerTo
}

type Writer[T any, PT Marshaler[T]] struct {
	Value PT
}

func NewWriter[T any, PT Marshaler[T]](val PT) Writer[T, PT] {
	return Writer[T, PT]{Value: val}
}

func (w Writer[T, PT]) MarshalJSONTo(enc *jsontext.Encoder) error {
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

// JSONUnmarshaler is a type constraint for decoding typed messages.
type Unmarshaler[T any] interface {
	*T
	json.UnmarshalerFrom
}

// Reader is a JSON decoder for typed messages. It should be used when the
// expected type is known at compile time.
type Reader[T any, PT Unmarshaler[T]] struct {
	Value PT
}

func NewReader[T any, PT Unmarshaler[T]]() Reader[T, PT] {
	return Reader[T, PT]{}
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

func (r *Reader[T, PT]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
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
	} else if r.Value, ok = val.(PT); !ok {
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
