// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Encoding represents the encoding used for the raw bytes of an Object,
// that is the original encoding used when the object was downloaded.
type Encoding int

const (
	ByteEncoding Encoding = iota
	JSONEncoding
)

// Object represents an object that has been downloded/crawled or read from a
// persistent store (after having been written to persistent storage following
// being crawled/downloaded). RawObject contains the raw bytes that the
// Object was created from and metadata.
type Object[T any] struct {
	Object T
	RawObject
}

func unmarshallByteEncoding(data []byte, obj any) error {
	switch obj := obj.(type) {
	case *[]byte:
		*obj = data
		return nil
	}
	return fmt.Errorf("byte encoding not compatibile for type: %T", obj)
}

func (o *Object[T]) Encode() ([]byte, error) {
	return o.RawObject.Encode()
}

func (o *Object[T]) Decode(data []byte) error {
	if err := o.RawObject.Decode(data); err != nil {
		return err
	}
	switch o.RawObject.Encoding {
	case JSONEncoding:
		return json.Unmarshal(o.RawObject.Bytes, &o.Object)
	case ByteEncoding:
		return unmarshallByteEncoding(o.RawObject.Bytes, &o.Object)
	}
	return fmt.Errorf("unsupported encoding used for the raw object: %v", o.RawObject.Encoding)
}

func (o *Object[T]) Write(wr io.Writer) error {
	return o.RawObject.Write(wr)
}

func (o *Object[T]) Read(rd io.Reader) error {
	if err := o.RawObject.Read(rd); err != nil {
		return err
	}
	switch o.RawObject.Encoding {
	case JSONEncoding:
		return json.Unmarshal(o.RawObject.Bytes, &o.Object)
	case ByteEncoding:
		return unmarshallByteEncoding(o.RawObject.Bytes, &o.Object)
	}
	return fmt.Errorf("unsupported encoding used for the raw object: %v", o.RawObject.Encoding)
}

// RawObject represents the raw data for an Object, that is, the original
// bytes the Object was created from and metadata such as its content.Type,
// original encoding, when it was created, any errors encountered when it was
// crawled/downloaded etc.
type RawObject struct {
	Type       Type
	CreateTime time.Time
	Encoding   Encoding
	Bytes      []byte
	Error      string
}

func (o *RawObject) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(o); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (o *RawObject) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(o)
}

func (o *RawObject) Write(wr io.Writer) error {
	return gob.NewEncoder(wr).Encode(o)
}

func (o *RawObject) Read(rd io.Reader) error {
	return gob.NewDecoder(rd).Decode(o)
}
