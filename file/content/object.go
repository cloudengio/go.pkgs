// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"cloudeng.io/file/content/internal"
)

// Object represents the result of an object/file download/crawl operation. As such
// it contains both the value that was downloaded and the result of the operation.
// The Value field represents the typed value of the result of the download
// or API operation. The Response field is the actual response for the download,
// API call and may include additional metadata such as object size, ownership
// etc. The content.Type is used by a reader to determine the type of the Value
// and Response fields and in this sense an object is a union.
//
// Objects are intended to be serialized to/from storage with the reader
// needing to determine the types of the serialized Value and Response from the
// encoded data. The format chosen for the serialized data is intended to allow
// for dealing with the different sources of both Value and Response and allows
// for each to be encoded using a different encoding. For example, a response
// from a rest API may be only encodable as json. Similarly responses generated
// by native go code are likely most conveniently encoded as gob.
// The serialized format is:
//
//	type []byte
//	valueEncoding uint8
//	responseEncoding uint8
//	value []byte
//	response []byte
//
// The gob format assumes that the decoder knows the type of the previously
// encoded binary data, including interface types. JSON cannot readily
// handle interface types.
//
// When gob encoding is used care must be taken to ensure that any
// fields that are interface types are appropriately registered with the
// gob package. error is a common such case and the Error function can be
// used to replace the existing error with a wrapper that implements the
// error interface and is registered with the gob package. Canonical usage
// is:
//
//	response.Err = content.Error(object.Err)
type Object[Value, Response any] struct {
	Type     Type
	Value    Value
	Response Response
}

// ObjectEncoding represents the encoding to be used for the object's Value
// and Response fields.
type ObjectEncoding int

const (
	InvalidObjectEncoding ObjectEncoding = iota
	GOBObjectEncoding                    = iota
	JSONObjectEncoding
)

// Encode encodes the object using the requested encodings.
func (o *Object[V, R]) Encode(valueEncoding, responseEncoding ObjectEncoding) ([]byte, error) {
	buf := bytes.Buffer{}
	if err := internal.WriteSlice(&buf, []byte(o.Type)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint8(valueEncoding)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint8(responseEncoding)); err != nil {
		return nil, err
	}
	var err error
	var vbuf, rbuf bytes.Buffer
	switch valueEncoding {
	case GOBObjectEncoding:
		err = gob.NewEncoder(&vbuf).Encode(o.Value)
	case JSONObjectEncoding:
		err = json.NewEncoder(&vbuf).Encode(o.Value)
	default:
		return nil, fmt.Errorf("unsupported value encoding: %v", valueEncoding)
	}
	if err != nil {
		return nil, err
	}
	switch responseEncoding {
	case GOBObjectEncoding:
		err = gob.NewEncoder(&rbuf).Encode(o.Response)
	case JSONObjectEncoding:
		err = json.NewEncoder(&rbuf).Encode(o.Response)
	default:
		return nil, fmt.Errorf("unsupported response encoding: %v", responseEncoding)
	}
	if err != nil {
		return nil, err
	}
	if err := internal.WriteSlice(&buf, vbuf.Bytes()); err != nil {
		return nil, err
	}
	if err := internal.WriteSlice(&buf, rbuf.Bytes()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes the object in data.
func (o *Object[V, R]) Decode(data []byte) error {
	rd := bytes.NewReader(data)
	data, err := internal.ReadSlice(rd)
	if err != nil {
		return err
	}
	o.Type = Type(data)
	var valueEncoding, responseEncoding uint8
	if err := binary.Read(rd, binary.LittleEndian, &valueEncoding); err != nil {
		return err
	}
	if err := binary.Read(rd, binary.LittleEndian, &responseEncoding); err != nil {
		return err
	}
	vbuf, err := internal.ReadSlice(rd)
	if err != nil {
		return err
	}
	rbuf, err := internal.ReadSlice(rd)
	if err != nil {
		return err
	}
	switch valueEncoding {
	case GOBObjectEncoding:
		err = gob.NewDecoder(bytes.NewBuffer(vbuf)).Decode(&o.Value)
	case JSONObjectEncoding:
		err = json.NewDecoder(bytes.NewBuffer(vbuf)).Decode(&o.Value)
	default:
		return fmt.Errorf("unsupported value encoding: %v", valueEncoding)
	}
	if err != nil {
		return err
	}
	switch responseEncoding {
	case GOBObjectEncoding:
		err = gob.NewDecoder(bytes.NewBuffer(rbuf)).Decode(&o.Response)
	case JSONObjectEncoding:
		err = json.NewDecoder(bytes.NewBuffer(rbuf)).Decode(&o.Response)
	default:
		return fmt.Errorf("unsupported value encoding: %v", responseEncoding)
	}
	return err
}

// Error is an implementation of error that is registered with the gob
// package and marshals the error as the string value returned by its Error()
// method. It will return nil if the specified error is nil. Common usage
// is:
//
//	response.Err = content.Error(object.Err)
func Error(err error) error {
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
