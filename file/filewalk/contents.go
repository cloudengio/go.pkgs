// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"encoding/binary"
	"io/fs"
)

type Entry struct {
	Name string
	Type fs.FileMode // Type is the Type portion of fs.FileMode
}

func (de Entry) IsDir() bool {
	return de.Type.IsDir()
}

type EntryList []Entry

func appendString(buf []byte, s string) []byte {
	buf = binary.AppendVarint(buf, int64(len(s)))
	return append(buf, s...)
}

func decodeString(data []byte) (int, string) {
	l, n := binary.Varint(data)
	return n + int(l), string(data[n : n+int(l)])
}

// AppendBinary appends a binary encoded instance of Info to the supplied
// byte slice.
func (el EntryList) AppendBinary(data []byte) ([]byte, error) {
	data = binary.AppendVarint(data, int64(len(el)))
	var err error
	for _, c := range el {
		data = appendString(data, c.Name)
		data = binary.LittleEndian.AppendUint32(data, uint32(c.Type))
	}
	return data, err
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (el EntryList) MarshalBinary() ([]byte, error) {
	return el.AppendBinary(make([]byte, 0, 200))
}

// DecodeBinary decodes the supplied data into an InfoList and returns
// the remaining data.
func (el *EntryList) DecodeBinary(data []byte) ([]byte, error) {
	l, n := binary.Varint(data)
	data = data[n:]
	*el = make(EntryList, l)
	for i := range *el {
		n, (*el)[i].Name = decodeString(data)
		data = data[n:]
		(*el)[i].Type = fs.FileMode(binary.LittleEndian.Uint32(data))
		data = data[4:]
	}
	return data, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (el *EntryList) UnmarshalBinary(data []byte) (err error) {
	_, err = el.DecodeBinary(data)
	return
}
