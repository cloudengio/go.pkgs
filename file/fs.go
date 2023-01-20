// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"io/fs"
	"time"
)

func init() {
	gob.Register(&Info{})
}

// FS extends fs.FS with OpenCtx.
type FS interface {
	fs.FS
	OpenCtx(ctx context.Context, name string) (fs.File, error)
}

// WrapFS wraps an fs.FS to implement file.FS.
func WrapFS(fs fs.FS) FS {
	return &fsFromFS{fs}
}

type fsFromFS struct {
	fs.FS
}

func (f *fsFromFS) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	return f.Open(name)
}

// Info implements fs.FileInfo with gob and json encoding/decoding. Note that
// the Sys value is not encoded/decode and is only avalilable within the
// process that originally created the info Instance.
type Info struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sysInfo interface{}
}

// NewInfo creates a new instance of Info.
func NewInfo(name string, size int64, mode fs.FileMode, mod time.Time, dir bool, sys interface{}) *Info {
	return &Info{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: mod,
		isDir:   dir,
		sysInfo: sys,
	}
}

func (fi *Info) Name() string {
	return fi.name
}

func (fi *Info) Size() int64 {
	return fi.size
}

func (fi *Info) Mode() fs.FileMode {
	return fi.mode
}

func (fi *Info) ModTime() time.Time {
	return fi.modTime
}

func (fi *Info) IsDir() bool {
	return fi.isDir
}

func (fi *Info) Sys() interface{} {
	return fi.sysInfo
}

func (fi *Info) asInfo() info {
	return info{
		Filename:    fi.name,
		FileSize:    fi.size,
		FileMode:    fi.mode,
		FileModTime: fi.modTime,
		FileIsDir:   fi.isDir,
	}
}

func (fi *Info) fromInfo(i info) {
	fi.name = i.Filename
	fi.size = i.FileSize
	fi.mode = i.FileMode
	fi.modTime = i.FileModTime
	fi.isDir = i.FileIsDir
}

func (fi *Info) GobEncode() ([]byte, error) {
	tmp := fi.asInfo()
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(tmp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (fi *Info) MarshalJSON() ([]byte, error) {
	return json.Marshal(fi.asInfo())
}

func (fi *Info) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var tmp info
	if err := dec.Decode(&tmp); err != nil {
		return err
	}
	fi.fromInfo(tmp)
	return nil
}

func (fi *Info) UnmarshalJSON(data []byte) error {
	var tmp info
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	fi.fromInfo(tmp)
	return nil
}

// info is like Info but without the Sys field.
type info struct {
	Filename    string
	FileSize    int64
	FileMode    fs.FileMode
	FileModTime time.Time
	FileIsDir   bool
}
