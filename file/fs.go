// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/fs"
	"time"
)

func init() {
	gob.Register(&Info{})
}

// FS extends fs.FS with Scheme and OpenCtx.
type FS interface {
	fs.FS

	// Scheme returns the URI scheme that this FS supports. Scheme should
	// be "file" for local file system access.
	Scheme() string

	// OpenCtx is like fs.Open but with a context.
	OpenCtx(ctx context.Context, name string) (fs.File, error)
}

// WrapFS wraps an fs.FS to implement file.FS.
func WrapFS(fs fs.FS) FS {
	return &fsFromFS{fs}
}

type fsFromFS struct {
	fs.FS
}

// Scheme returns the URI scheme that this FS supports, which in for an fs.FS
// is always "file".
func (f *fsFromFS) Scheme() string {
	return "file"
}

// OpenCtx just calls fs.Open.
func (f *fsFromFS) OpenCtx(_ context.Context, name string) (fs.File, error) {
	return f.Open(name)
}

var _ fs.FileInfo = (*Info)(nil)

// Info implements fs.FileInfo to provide binary, gob and json encoding/decoding.
// The SysInfo field is not encoded/decoded and hence is only available for use
// within the process that Info was instantiated in.
type Info struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sysInfo any
}

// NewInfo creates a new instance of Info.
func NewInfo(
	name string,
	size int64,
	mode fs.FileMode,
	modTime time.Time,
	sysInfo any) Info {
	return Info{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
		sysInfo: sysInfo,
	}
}

func NewInfoFromFileInfo(fi fs.FileInfo) Info {
	return NewInfo(
		fi.Name(),
		fi.Size(),
		fi.Mode(),
		fi.ModTime(),
		fi.Sys())
}

// Name implements fs.FileInfo.
func (fi Info) Name() string {
	return fi.name
}

// Size implements fs.FileInfo.
func (fi Info) Size() int64 {
	return fi.size
}

// Mode implements fs.FileInfo.
func (fi Info) Mode() fs.FileMode {
	return fi.mode
}

// ModTime implements fs.FileInfo.
func (fi Info) ModTime() time.Time {
	return fi.modTime
}

// IsDir implements fs.FileInfo.
func (fi Info) IsDir() bool {
	return fi.mode.IsDir()
}

// Sys implements fs.FileInfo.
func (fi Info) Sys() any {
	return fi.sysInfo
}

func (fi *Info) SetSys(i any) {
	fi.sysInfo = i
}

// info is like Info but without the Sys field.
type info struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    fs.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
}

func (fi Info) MarshalJSON() ([]byte, error) {
	return json.Marshal(info{
		Name:    fi.name,
		Size:    fi.size,
		Mode:    fi.mode,
		ModTime: fi.modTime,
	})
}

func (fi *Info) UnmarshalJSON(data []byte) error {
	var tmp info
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	fi.name = tmp.Name
	fi.size = tmp.Size
	fi.mode = tmp.Mode
	fi.modTime = tmp.ModTime
	return nil
}

func appendString(buf []byte, s string) []byte {
	buf = binary.AppendVarint(buf, int64(len(s)))
	return append(buf, s...)
}

func decodeString(data []byte) (int, string) {
	l, n := binary.Varint(data)
	return n + int(l), string(data[n : n+int(l)])
}

func (fi *Info) AppendBinary(data []byte) ([]byte, error) {
	data = append(data, 0x1)                                       // version
	data = appendString(data, fi.name)                             // name
	data = binary.AppendVarint(data, fi.size)                      // size
	data = binary.LittleEndian.AppendUint32(data, uint32(fi.mode)) // filemode
	out, err := fi.modTime.MarshalBinary()                         // modtime
	if err != nil {
		return nil, err
	}
	data = binary.AppendVarint(data, int64(len(out)))
	data = append(data, out...)
	return data, nil
}

// Implements encoding.BinaryMarshaler.
func (fi Info) MarshalBinary() ([]byte, error) {
	return fi.AppendBinary(make([]byte, 0, 100))
}

// Implements encoding.BinaryUnmarshaler.
func (fi *Info) UnmarshalBinary(data []byte) error {
	_, err := fi.DecodeBinary(data)
	return err
}

// DecodeBinary decodes the supplied data into the receiver and returns
// the remaining data.
func (fi *Info) DecodeBinary(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("file.Info: insufficient data")
	}
	if data[0] != 0x1 {
		return nil, fmt.Errorf("file.Info: invalid version of binary encoding: got %x, want %x", data[0], 0x1)
	}
	data = data[1:] // version
	var n int
	n, fi.name = decodeString(data) // name
	data = data[n:]
	fi.size, n = binary.Varint(data) // size
	data = data[n:]

	fi.mode = fs.FileMode(binary.LittleEndian.Uint32(data)) // filemode
	data = data[4:]

	ts, n := binary.Varint(data) // modtime
	data = data[n:]
	if err := fi.modTime.UnmarshalBinary(data[0:ts]); err != nil {
		return nil, err
	}
	data = data[ts:]
	return data, nil
}

// InfoList represents a list of Info instances. It provides efficient
// encoding/decoding operations.
type InfoList []Info

// Append appends an Info instance to the list and returns the updated list.
func (il InfoList) AppendInfo(info Info) InfoList {
	return append(il, Info{
		name:    info.name,
		size:    info.size,
		mode:    info.mode,
		modTime: info.modTime,
		sysInfo: info.sysInfo,
	})
}

// AppendBinary appends a binary encoded instance of Info to the supplied
// byte slice.
func (il InfoList) AppendBinary(data []byte) ([]byte, error) {
	data = binary.AppendVarint(data, int64(len(il)))
	var err error
	for _, c := range il {
		data, err = c.AppendBinary(data)
		if err != nil {
			return nil, err
		}
	}
	return data, err
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (il InfoList) MarshalBinary() ([]byte, error) {
	return il.AppendBinary(make([]byte, 0, 200))
}

// DecodeBinaryInfoList decodes the supplied data into an InfoList and returns
// the remaining data.
func DecodeBinaryInfoList(data []byte) (InfoList, []byte, error) {
	l, n := binary.Varint(data)
	data = data[n:]
	il := make(InfoList, l)
	for i := range il {
		var err error
		data, err = il[i].DecodeBinary(data)
		if err != nil {
			return nil, nil, err
		}
	}
	return il, data, nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (il *InfoList) UnmarshalBinary(data []byte) (err error) {
	*il, _, err = DecodeBinaryInfoList(data)
	return
}
