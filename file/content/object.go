// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Object represents an object/file that has been downloaded/crawled or is the
// result of an API invocation. The Value field represents the typed value
// of the result of the download or API operation. The Response field
// is the actual response for the download, API call etc. The Response
// may include additional metadata.
//
// Object supports encoding/decoding either in binary or gob format.
// The gob format assumes that the decoder knows the type of the previously
// encoded binary. The binary format encodes the content.Type and a byte
// slice in separately. This allows for reading the encoded data without
// necessarily knowing the type of the encoded object.
//
// When gob encoding is supported care must be taken to ensure that any
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

// Encode encodes the object using gob.
func (o *Object[V, R]) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(o); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes the object in data using gob.
func (o *Object[V, R]) Decode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(o)
}

// Write encodes the object using gob to the specified writer.
func (o *Object[V, R]) Write(wr io.Writer) error {
	return gob.NewEncoder(wr).Encode(o)
}

// Read decodes the object using gob from the specified reader.
func (o *Object[V, R]) Read(rd io.Reader) error {
	return gob.NewDecoder(rd).Decode(o)
}

// EncodeBinary will encode the object using the binary encoding format.
func (o *Object[V, R]) EncodeBinary(wr io.Writer) error {
	data, err := o.Encode()
	if err != nil {
		return err
	}
	return EncodeBinary(wr, o.Type, data)
}

// DecodeBinary will decode the object using the binary encoding format.
func (o *Object[V, R]) DecodeBinary(rd io.Reader) error {
	ctype, data, err := DecodeBinary(rd)
	if err != nil {
		return err
	}
	if err := o.Decode(data); err != nil {
		if ctype != o.Type {
			return fmt.Errorf("content types not match: %v != %v", ctype, o.Type)
		}
	}
	return nil
}

// WriteObjectBinary will encode the object using the binary encoding format
// to the specified file.
func (o *Object[V, R]) WriteObjectBinary(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	wr, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return o.EncodeBinary(wr)
}

// ReadObjectBinary will decode the object using the binary encoding format
// from the specified file.
func (o *Object[V, R]) ReadObjectBinary(path string) error {
	rd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer rd.Close()
	return o.DecodeBinary(rd)
}

// WriteObject will encode the object using gob to the specified file.
func (o *Object[V, R]) WriteObject(path string) error {
	wr, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer wr.Close()
	return o.Write(wr)
}

func (o *Object[V, R]) ReadObject(path string) error {
	rd, err := os.Open(path)
	if err != nil {
		return err
	}
	defer rd.Close()
	return o.Read(rd)
}

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

// EncodeBinary encodes the specified content.Type and byte slice as binary data.
func EncodeBinary(wr io.Writer, ctype Type, data []byte) error {
	if err := writeSlice(wr, []byte(ctype)); err != nil {
		return err
	}
	return writeSlice(wr, data)
}

// DecodeBinary decodes the result of a previous call to EncodeBinary.
func DecodeBinary(rd io.Reader) (ctype Type, data []byte, err error) {
	tmp, err := readSlice(rd)
	if err != nil {
		return "", nil, err
	}
	ctype = Type(string(tmp))
	tmp, err = readSlice(rd)
	if err != nil {
		return "", nil, err
	}
	data = tmp
	return
}

// WriteBinary writes the results of EncodeBinary(cytpe, data) to the
// specified file.
func WriteBinary(path string, ctype Type, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	wr, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer wr.Close()
	return EncodeBinary(wr, ctype, data)
}

// ReadBinary reads the contents of path and interprets them using
// DecodeBinary.
func ReadBinary(path string) (ctype Type, data []byte, err error) {
	rd, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer rd.Close()
	return DecodeBinary(rd)
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
