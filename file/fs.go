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
func (f *fsFromFS) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	return f.Open(name)
}

// Info extends fs.FileInfo to provide additional information such as
// user/group, symbolic link status etc, as well gob and json encoding/decoding.
// Note that the Sys value is not encoded/decoded and is only avalilable within
// the process that originally created the info Instance.
type Info struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	isLink  bool
	user    string
	group   string
	sysInfo interface{}
}

// InfoOption is used to provide additional fields when creating
// an Info instance using NewInfo.
type InfoOption struct {
	ModTime time.Time
	User    string
	Group   string
	IsDir   bool
	IsLink  bool
	SysInfo interface{}
}

// NewInfo creates a new instance of Info.
func NewInfo(name string, size int64, mode fs.FileMode, options InfoOption) Info {
	return Info{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: options.ModTime,
		isDir:   options.IsDir,
		isLink:  options.IsLink,
		user:    options.User,
		group:   options.Group,
		sysInfo: options.SysInfo,
	}
}

// Name implements fs.FileInfo.
func (fi *Info) Name() string {
	return fi.name
}

// Size implements fs.FileInfo.
func (fi *Info) Size() int64 {
	return fi.size
}

// Mode implements fs.FileInfo.
func (fi *Info) Mode() fs.FileMode {
	return fi.mode
}

// ModTime implements fs.FileInfo.
func (fi *Info) ModTime() time.Time {
	return fi.modTime
}

// IsDir implements fs.FileInfo.
func (fi *Info) IsDir() bool {
	return fi.isDir
}

// Sys implements fs.FileInfo.
func (fi *Info) Sys() interface{} {
	return fi.sysInfo
}

// User returns the user associated with the file.
func (fi *Info) User() string {
	return fi.user
}

// Group returns the group associated with the file.
func (fi *Info) Group() string {
	return fi.group
}

// IsLink returns true if the file is a symbolic link.
func (fi *Info) IsLink() bool {
	return fi.isLink
}

// info is like Info but without the Sys field.
type info struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    fs.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	IsDir   bool        `json:"isDir"`
	IsLink  bool        `json:"isLink"`
	User    string      `json:"user"`
	Group   string      `json:"group"`
}

func (fi *Info) asInfo() info {
	return info{
		Name:    fi.name,
		Size:    fi.size,
		Mode:    fi.mode,
		ModTime: fi.modTime,
		IsDir:   fi.isDir,
		IsLink:  fi.isLink,
		User:    fi.user,
		Group:   fi.group,
	}
}

func (fi *Info) fromInfo(i info) {
	fi.name = i.Name
	fi.size = i.Size
	fi.mode = i.Mode
	fi.modTime = i.ModTime
	fi.isDir = i.IsDir
	fi.isLink = i.IsLink
	fi.user = i.User
	fi.group = i.Group
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
