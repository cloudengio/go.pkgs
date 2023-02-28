// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"bytes"
	"encoding/gob"
	"io"
)

// Object represents an object/file that has been downloaded/crawled or is the
// result of an API invocation. The Value field represents the typed value
// of the result of the download or API operation. The Response field
// is the actual response for the download, API call etc. The Response
// may include additional metadata.
//
// Object should be used whenever generic operations over either downloaded
// content or API responses are required. Gob encoding is supported, but
// care must be taken to ensure that any fields that are interface types
// are appropriately registered with the gob package. error is a common
// case and the GobError function can be used to replace the existing error
// with a wrapper that implements the error interface and is registered
// with the gob package. Canonical usage is:
//
//	response.Err = content.GobError(object.Err)
type Object[Value, Response any] struct {
	Value    Value
	Response Response
	Type     Type
}

func (o *Object[V, R]) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(o); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (o *Object[V, R]) Decode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(o)
}

func (o *Object[V, R]) Write(wr io.Writer) error {
	return gob.NewEncoder(wr).Encode(o)
}

func (o *Object[V, R]) Read(rd io.Reader) error {
	return gob.NewDecoder(rd).Decode(o)
}

// GobError is an implementation of error that is registered with the gob
// package and marshals the error as the string value returned by its Error()
// method. It will return nil if the specified error is nil. Common usage
// is:
//
//	response.Err = content.GobError(object.Err)
func GobError(err error) error {
	if err == nil {
		return nil
	}
	return &errorString{Err: err.Error()}
}

type errorString struct{ Err string }

func (e *errorString) Error() string { return e.Err }

func init() {
	gob.Register(&errorString{})
}
