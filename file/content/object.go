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
	"io"
	"os"
	"path/filepath"
)

// Object represents the result of object/file download/crawl operation. As such
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

func writeSlice(wr io.Writer, data []byte) error {
	if err := binary.Write(wr, binary.LittleEndian, int64(len(data))); err != nil {
		return err
	}
	return binary.Write(wr, binary.LittleEndian, data)
}

const limit = 1 << 23 // 8MB seems large enough

func readSlice(rd io.Reader) ([]byte, error) {
	var l int64
	if err := binary.Read(rd, binary.LittleEndian, &l); err != nil {
		return nil, err
	}
	if l > limit {
		return nil, fmt.Errorf("data size too large (%v > %v): likely the file is in the wrong format", l, limit)
	}
	data := make([]byte, l)
	if err := binary.Read(rd, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	return data, nil
}

// Encode encodes the object using the requested encodings.
func (o *Object[V, R]) Encode(valueEncoding, responseEncoding ObjectEncoding) ([]byte, error) {
	buf := bytes.Buffer{}
	if err := writeSlice(&buf, []byte(o.Type)); err != nil {
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
	if err := writeSlice(&buf, vbuf.Bytes()); err != nil {
		return nil, err
	}
	if err := writeSlice(&buf, rbuf.Bytes()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes the object in data.
func (o *Object[V, R]) Decode(data []byte) error {
	rd := bytes.NewReader(data)
	data, err := readSlice(rd)
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
	vbuf, err := readSlice(rd)
	if err != nil {
		return err
	}
	rbuf, err := readSlice(rd)
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

// WriteObject will encode the object using the requested encoding to the
// specified file. It will create the directory that the file is to be written
// to if it does not exist.
func (o *Object[V, R]) WriteObjectFile(path string, valueEncoding, responseEncoding ObjectEncoding) error {
	buf, err := o.Encode(valueEncoding, responseEncoding)
	if err != nil {
		return err
	}
	wr, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// Try to create the directory that the file is to be written to.
		os.MkdirAll(filepath.Dir(path), 0700) //nolint:errcheck
		wr, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
	}
	defer wr.Close()
	_, err = wr.Write(buf)
	return err
}

// ReadObjectFile will read the specified file and return the object type, encoding and the
// the contents of that file. The returned byte slice can be used to decode the object using
// its Decode method.
func ReadObjectFile(path string) (Type, []byte, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	rd := bytes.NewReader(buf)
	data, err := readSlice(rd)
	if err != nil {
		return "", nil, err
	}
	return Type(data), buf, err
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
