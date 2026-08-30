// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
)

// outer carries a typed message as one of its fields and implements the
// json/v2 interfaces itself, so it can use Decode and Writer directly rather
// than holding a Reader. Inner is the payload type itself, not a wrapper.
type outer struct {
	Name  string
	Inner payload
	Count int
}

func (o *outer) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if tok, err := dec.ReadToken(); err != nil {
		return err
	} else if tok.Kind() != jsontext.KindBeginObject {
		return fmt.Errorf("expected '{', got %v", tok.Kind())
	}
	for {
		tok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		if tok.Kind() == jsontext.KindEndObject {
			return nil
		}
		switch tok.String() {
		case "inner":
			// Decode consumes exactly the typed message, leaving the decoder
			// positioned on the member that follows it.
			if err := jsonpayload.Decode(dec, &o.Inner); err != nil {
				return err
			}
		case "name":
			if err := decodeMember(dec, &o.Name); err != nil {
				return err
			}
		case "count":
			if err := decodeMember(dec, &o.Count); err != nil {
				return err
			}
		default:
			if _, err := dec.ReadValue(); err != nil {
				return err
			}
		}
	}
}

func decodeMember[T any](dec *jsontext.Decoder, into *T) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	return json.Unmarshal(v, into)
}

func (o *outer) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("name")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(o.Name)); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("inner")); err != nil {
		return err
	}
	// Writer has a value receiver, so the typed message can be written
	// inline without keeping a wrapper around.
	if err := jsonpayload.NewWriter(&o.Inner).MarshalJSONTo(enc); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("count")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Int(int64(o.Count))); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// TestDecodeWithinEnclosingUnmarshaler verifies that a type which implements
// UnmarshalJSONFrom itself can call Decode directly for an embedded typed
// message, with no Reader involved. The member following the message is only
// decoded correctly if Decode consumes exactly the message's tokens.
func TestDecodeWithinEnclosingUnmarshaler(t *testing.T) {
	inner, err := json.Marshal(jsonpayload.NewWriter(&payload{A: 42}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	doc := fmt.Sprintf(`{"name":"outer","inner":%s,"count":7}`, inner)

	var o outer
	if err := json.Unmarshal([]byte(doc), &o); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := o.Name, "outer"; got != want {
		t.Errorf("Name: got %q, want %q", got, want)
	}
	if got, want := o.Inner, (payload{A: 42}); got != want {
		t.Errorf("Inner: got %+v, want %+v", got, want)
	}
	if got, want := o.Count, 7; got != want {
		t.Errorf("Count: got %v, want %v; the decoder was left mispositioned", got, want)
	}
}

// TestWriterWithinEnclosingMarshaler is the encoding counterpart: a typed
// message written inline by an enclosing marshaler round trips through the
// enclosing unmarshaler.
func TestWriterWithinEnclosingMarshaler(t *testing.T) {
	in := outer{Name: "outer", Inner: payload{A: 42}, Count: 7}
	buf, err := json.Marshal(&in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"name":"outer","inner":{"type":"` + testPkg +
		`.payload","payload":{"A":42}},"count":7}`
	if got := string(buf); got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}

	var out outer
	if err := json.Unmarshal(buf, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := out; got != in {
		t.Errorf("round trip: got %+v, want %+v", got, in)
	}
}
