// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// Wrapper adapts any json-marshalable type T to the json.MarshalerTo
// and json.UnmarshalerFrom interfaces. It is not intended for
// performance-sensitive code. Also note that it's type name will
// include the full generic type name, e.g. "cloudeng.io/encoding/json/jsonpayload.Wrapper[example.com/pkg.MyType]".
type Wrapper[T any] struct {
	Value T
}

func (j *Wrapper[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	b, err := json.Marshal(j.Value)
	if err != nil {
		return err
	}
	return enc.WriteValue(jsontext.Value(b))
}

func (j *Wrapper[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}
	return json.Unmarshal(val, &j.Value)
}
