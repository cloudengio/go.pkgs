// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonpayload"
	"cloudeng.io/types"
)

// TypeA has state with slice and string fields.
type TypeA struct {
	Name    string `json:"name"`
	Numbers []int  `json:"numbers"`
}

func (a *TypeA) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain TypeA
	b, err := json.Marshal((*plain)(a))
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(b))
}

func (a *TypeA) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain TypeA
	return json.Unmarshal(val, (*plain)(a))
}

// TypeB has completely different fields.
type TypeB struct {
	Count int  `json:"count"`
	Flag  bool `json:"flag"`
}

func (b *TypeB) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain TypeB
	buf, err := json.Marshal((*plain)(b))
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(buf))
}

func (b *TypeB) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain TypeB
	return json.Unmarshal(val, (*plain)(b))
}

// TypeConflictA has a field of map type.
type TypeConflictA struct {
	Data map[string]int `json:"data"`
}

func (c *TypeConflictA) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain TypeConflictA
	buf, err := json.Marshal((*plain)(c))
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(buf))
}

func (c *TypeConflictA) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain TypeConflictA
	return json.Unmarshal(val, (*plain)(c))
}

// TypeConflictB has the same field name as TypeConflictA, but with incompatible string type.
type TypeConflictB struct {
	Data string `json:"data"`
}

func (c *TypeConflictB) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plain TypeConflictB
	buf, err := json.Marshal((*plain)(c))
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(buf))
}

func (c *TypeConflictB) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	type plain TypeConflictB
	return json.Unmarshal(val, (*plain)(c))
}

func init() {
	jsonpayload.RegisterType[TypeA]()
	jsonpayload.RegisterType[TypeB]()
	jsonpayload.RegisterType[TypeConflictA]()
	jsonpayload.RegisterType[TypeConflictB]()
}

// TestMismatchedTypeDisjointFields tests the case where sender encodes TypeA
// but claims it is TypeB in the envelope. When decoded into TypeB, unknown fields
// from TypeA are safely ignored, leaving TypeB in its zero state without crash or corruption.
func TestMismatchedTypeDisjointFields(t *testing.T) {
	valA := &TypeA{Name: "secret_data", Numbers: []int{1, 2, 3, 4}}
	encodedA, err := json.Marshal(valA)
	if err != nil {
		t.Fatalf("Marshal TypeA: %v", err)
	}

	// Craft envelope claiming to be TypeB with TypeA's payload.
	spoofedEnvelope := fmt.Sprintf(`{"type":%q,"payload":%s}`, types.TypeName[TypeB](), encodedA)

	t.Run("ReaderAny", func(t *testing.T) {
		var rd jsonpayload.ReaderAny
		if err := json.Unmarshal([]byte(spoofedEnvelope), &rd); err != nil {
			t.Fatalf("Unmarshal into ReaderAny failed: %v", err)
		}
		if got, want := rd.Type, types.TypeName[TypeB](); got != want {
			t.Errorf("rd.Type: got %q, want %q", got, want)
		}
		b, ok := rd.Value.(*TypeB)
		if !ok {
			t.Fatalf("rd.Value is %T, want *TypeB", rd.Value)
		}
		// Fields of TypeB are untouched zero values.
		if b.Count != 0 || b.Flag != false {
			t.Errorf("TypeB fields corrupted: got %+v, want zero value", b)
		}
	})

	t.Run("ReadWriterAny", func(t *testing.T) {
		var rw jsonpayload.ReadWriterAny
		if err := json.Unmarshal([]byte(spoofedEnvelope), &rw); err != nil {
			t.Fatalf("Unmarshal into ReadWriterAny failed: %v", err)
		}
		if got, want := rw.Type, types.TypeName[TypeB](); got != want {
			t.Errorf("rw.Type: got %q, want %q", got, want)
		}
		b, ok := rw.Value.(*TypeB)
		if !ok {
			t.Fatalf("rw.Value is %T, want *TypeB", rw.Value)
		}
		if b.Count != 0 || b.Flag != false {
			t.Errorf("TypeB fields corrupted: got %+v, want zero value", b)
		}
	})

	t.Run("Reader", func(t *testing.T) {
		var target TypeB
		rd := jsonpayload.NewReader(&target)
		if err := json.Unmarshal([]byte(spoofedEnvelope), &rd); err != nil {
			t.Fatalf("Unmarshal into Reader failed: %v", err)
		}
		if target.Count != 0 || target.Flag != false {
			t.Errorf("target fields corrupted: got %+v, want zero value", target)
		}
	})
}

// TestMismatchedTypeIncompatibleJSONKinds tests when the payload is a completely
// incompatible JSON kind (e.g. array, string, boolean) for a struct type.
// It verifies that decoding returns an error gracefully without panics.
func TestMismatchedTypeIncompatibleJSONKinds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{"array payload", `[1, 2, 3, "extra"]`},
		{"string payload", `"just a string"`},
		{"number payload", `12345`},
		{"boolean payload", `true`},
		{"nested arrays", `[[1], [2], [3]]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spoofedEnvelope := fmt.Sprintf(`{"type":%q,"payload":%s}`, types.TypeName[TypeB](), tc.payload)

			var rd jsonpayload.ReaderAny
			err := json.Unmarshal([]byte(spoofedEnvelope), &rd)
			if err == nil {
				t.Fatalf("%v: expected unmarshal error for incompatible payload, got nil", tc.name)
			}

			var target TypeB
			r := jsonpayload.NewReader(&target)
			err = json.Unmarshal([]byte(spoofedEnvelope), &r)
			if err == nil {
				t.Fatalf("%v: expected Reader unmarshal error for incompatible payload, got nil", tc.name)
			}
		})
	}
}

// TestMismatchedTypeIncompatibleFields tests conflicting field types where TypeConflictA
// has data as a map, while TypeConflictB has data as a string.
// When decoding A's payload as B, it returns an error rather than corrupting memory.
func TestMismatchedTypeIncompatibleFields(t *testing.T) {
	valA := &TypeConflictA{Data: map[string]int{"alpha": 1, "beta": 2}}
	encodedA, err := json.Marshal(valA)
	if err != nil {
		t.Fatalf("Marshal TypeConflictA: %v", err)
	}

	spoofedEnvelope := fmt.Sprintf(`{"type":%q,"payload":%s}`, types.TypeName[TypeConflictB](), encodedA)

	t.Run("ReaderAny", func(t *testing.T) {
		var rd jsonpayload.ReaderAny
		err := json.Unmarshal([]byte(spoofedEnvelope), &rd)
		if err == nil {
			t.Fatal("expected error decoding map into string field, got nil")
		}
	})

	t.Run("ReadWriterAny", func(t *testing.T) {
		var rw jsonpayload.ReadWriterAny
		err := json.Unmarshal([]byte(spoofedEnvelope), &rw)
		if err == nil {
			t.Fatal("expected error decoding map into string field, got nil")
		}
	})

	t.Run("Reader", func(t *testing.T) {
		var target TypeConflictB
		rd := jsonpayload.NewReader(&target)
		err := json.Unmarshal([]byte(spoofedEnvelope), &rd)
		if err == nil {
			t.Fatal("expected error decoding map into string field, got nil")
		}
	})
}

// TestMismatchedTypeWrappedValues tests wrapped types when a primitive wrapper
// is claimed as a slice wrapper.
func TestMismatchedTypeWrappedValues(t *testing.T) {
	intWrapper := jsonpayload.Wrapper[int]{Value: 42}
	encoded, err := json.Marshal(&intWrapper)
	if err != nil {
		t.Fatalf("Marshal int Wrapper: %v", err)
	}

	sliceTypeName := jsonpayload.RegisterType[jsonpayload.Wrapper[[]string]]()
	spoofedEnvelope := fmt.Sprintf(`{"type":%q,"payload":%s}`, sliceTypeName, encoded)

	var rd jsonpayload.ReaderAny
	err = json.Unmarshal([]byte(spoofedEnvelope), &rd)
	if err == nil {
		t.Fatal("expected error decoding integer into string slice wrapper, got nil")
	}
}

// TestMismatchedTypeDirectDecodeMismatch tests that Decode reports a type mismatch
// immediately if the envelope type name does not match the expected type.
func TestMismatchedTypeDirectDecodeMismatch(t *testing.T) {
	valA := &TypeA{Name: "data", Numbers: []int{10}}
	encodedA, err := json.Marshal(jsonpayload.NewWriter(valA))
	if err != nil {
		t.Fatalf("Marshal TypeA: %v", err)
	}

	// Expected type is TypeB, but envelope contains TypeA.
	var targetB TypeB
	dec := jsontext.NewDecoder(strings.NewReader(string(encodedA)))
	err = jsonpayload.Decode(dec, &targetB)
	if err == nil {
		t.Fatal("expected type mismatch error from Decode, got nil")
	}
	if !strings.Contains(err.Error(), "expected type name") {
		t.Errorf("expected error mentioning 'expected type name', got %v", err)
	}
	// targetB remains unmutated.
	if targetB.Count != 0 || targetB.Flag != false {
		t.Errorf("targetB was mutated despite mismatch: %+v", targetB)
	}
}
