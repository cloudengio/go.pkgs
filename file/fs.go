// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"bytes"
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

	// Readlink returns the contents of a symbolic link.
	Readlink(ctx context.Context, path string) (string, error)

	// Stat will follow symlinks/redirects/aliases.
	Stat(ctx context.Context, path string) (Info, error)

	// Lstat will not follow symlinks/redirects/aliases.
	Lstat(ctx context.Context, path string) (Info, error)

	// Join is like filepath.Join for the filesystem supported by this filesystem.
	Join(components ...string) string

	// Base is like filepath.Base for the filesystem supported by this filesystem.
	Base(path string) string

	// IsPermissionError returns true if the specified error, as returned
	// by the filesystem's implementation, is a result of a permissions error.
	IsPermissionError(err error) bool

	// IsNotExist returns true if the specified error, as returned by the
	// filesystem's implementation, is a result of the object not existing.
	IsNotExist(err error) bool

	// XAttr returns extended attributes for the specified file.Info
	// and file.
	XAttr(ctx context.Context, path string, fi Info) (XAttr, error)

	// SysXAttr returns a representation of the extended attributes using the
	// native data type of the underlying file system. If existing is
	// non-nil and is of that file-system specific type the contents of
	// XAttr are merged into it.
	SysXAttr(existing any, merge XAttr) any
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

// NewInfoFromFileInfo creates a new instance of Info from a fs.FileInfo.
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

// Type implements fs.Entry
func (fi Info) Type() fs.FileMode {
	return fi.mode.Type()
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

// SetSys sets the SysInfo field.
func (fi *Info) SetSys(i any) {
	fi.sysInfo = i
}

// XAttr represents extended information about a directory or file.
type XAttr struct {
	UID, GID       uint64
	Device, FileID uint64
	Blocks         int64
	Hardlinks      uint64
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

func appendString(buf *bytes.Buffer, s string) {
	var storage [5]byte
	n := binary.PutVarint(storage[:], int64(len(s)))
	buf.Write(storage[:n])
	buf.WriteString(s)
}

func decodeString(data []byte) (int, string) {
	l, n := binary.Varint(data)
	return n + int(l), string(data[n : n+int(l)])
}

func (fi *Info) AppendBinary(buf *bytes.Buffer) error {
	buf.WriteByte(0x1)         // version
	appendString(buf, fi.name) // name
	var storage [128]byte
	data := storage[:0]
	data = binary.AppendVarint(data, fi.size)                      // size
	data = binary.LittleEndian.AppendUint32(data, uint32(fi.mode)) // filemode
	out, err := fi.modTime.MarshalBinary()                         // modtime
	if err != nil {
		return err
	}
	data = binary.AppendVarint(data, int64(len(out)))
	data = append(data, out...)
	buf.Write(data)
	return nil
}

// Implements encoding.BinaryMarshaler.
func (fi Info) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(100)
	if err := fi.AppendBinary(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
func (il InfoList) AppendBinary(buf *bytes.Buffer) error {
	var data [5]byte
	n := binary.PutVarint(data[:], int64(len(il)))
	buf.Write(data[:n])
	for _, c := range il {
		if err := c.AppendBinary(buf); err != nil {
			return err
		}
	}
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (il InfoList) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(1000)
	if err := il.AppendBinary(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
