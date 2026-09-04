// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdjson

import (
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"time"
)

// RFC3339Time is a time.Time that marshals to and from RFC3339 format.
type RFC3339Time time.Time

func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tt, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*t = RFC3339Time(tt)
	return nil
}

func (t RFC3339Time) String() string {
	return time.Time(t).Format(time.RFC3339)
}

// MarshalJSONTo implements json.MarshalerTo from encoding/json/v2, writing the
// time directly to the encoder rather than through an intermediate value.
func (t RFC3339Time) MarshalJSONTo(enc *jsontext.Encoder) error {
	return enc.WriteToken(jsontext.String(time.Time(t).Format(time.RFC3339)))
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom from encoding/json/v2. A
// null leaves the value unchanged, as it does for UnmarshalJSON.
func (t *RFC3339Time) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch tok.Kind() {
	case jsontext.KindNull:
		return nil
	case jsontext.KindString:
		parsed, err := time.Parse(time.RFC3339, tok.String())
		if err != nil {
			return err
		}
		*t = RFC3339Time(parsed)
		return nil
	}
	return fmt.Errorf("expected a string for a time, got %v", tok.Kind())
}
