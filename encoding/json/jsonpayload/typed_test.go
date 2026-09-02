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

type myStruct struct {
	A int `json:"A"`
}

func (a *myStruct) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteValue(jsontext.Value(fmt.Sprintf(`{"A":%d}`, a.A)))
}

func (a *myStruct) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	var tmp struct {
		A int `json:"A"`
	}
	if err := json.Unmarshal([]byte(val), &tmp); err != nil {
		return err
	}
	a.A = tmp.A
	return nil
}

func init() {
	jsonpayload.RegisterType[myStruct]()
}

func TestEncodeDecode(t *testing.T) {
	val := myStruct{A: 42}
	pl := jsonpayload.NewWriter(&val)

	buf, err := json.Marshal(pl)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	rd := jsonpayload.NewReader(&myStruct{})
	if err := json.Unmarshal(buf, &rd); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if rd.Value.A != val.A {
		t.Fatalf("Decoded value mismatch: got %v, want %v", rd.Value.A, val.A)
	}

	rda := jsonpayload.ReaderAny{Value: &myStruct{}}
	if err := json.Unmarshal(buf, &rda); err != nil {
		t.Fatalf("Unmarshal into ReaderAny failed: %v", err)
	}
	if rda.Value.(*myStruct).A != val.A {
		t.Fatalf("Decoded value mismatch in ReaderAny: got %v, want %v", rda.Value.(*myStruct).A, val.A)
	}
}
