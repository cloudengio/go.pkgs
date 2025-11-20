// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
)

// NewErrorKeyNotFound creates a new Error indicating that the specified key
// was not found that is compatible with errors.Is and ErrorKeyNotFound.
func NewErrorKeyNotFound(keyname string) *Error {
	return &Error{
		Message: "key not found",
		Detail:  keyname,
	}
}

// ErrKeyNotFound can be used as the target of errors.Is to check for a
// key not found error.
var ErrKeyNotFound = NewErrorKeyNotFound("")

// Error represents an error returned by a plugin.
type Error struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.Detail)
}

func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}
	var err *Error
	if errors.As(target, &err) {
		return e.Message == err.Message
	}
	return false
}

// Request represents the request to the keychain plugin.
type Request struct {
	ID          int32           `json:"id,omitempty"`
	Keyname     string          `json:"keyname"`
	Write       bool            `json:"write,omitempty"`
	Contents    string          `json:"contents,omitempty"` // base64 encoded
	SysSpecific json.RawMessage `json:"sys_specific,omitempty"`
}

// Response represents the response from the keychain plugin.
type Response struct {
	ID          int32           `json:"id,omitempty"`
	Contents    string          `json:"contents"` // base64 encoded
	Error       *Error          `json:"error,omitempty"`
	SysSpecific json.RawMessage `json:"sys_specific,omitempty"`
}

var nextID int32 = 1

func NextID() int32 {
	return atomic.AddInt32(&nextID, 1)
}

// NewRequest creates a Request to read a key with the given keyname and
// system-specific data.
// The ID is automatically generated and is unique for each call to this
// function.
func NewRequest(keyname string, sysSpecific any) (Request, error) {
	var sysSpecificJSON json.RawMessage
	if sysSpecific != nil {
		b, err := json.Marshal(sysSpecific)
		if err != nil {
			return Request{}, err
		}
		sysSpecificJSON = b
	}
	return Request{
		ID:          NextID(),
		Keyname:     keyname,
		SysSpecific: sysSpecificJSON,
	}, nil
}

// NewWriteRequest creates a Request to write a key with the given keyname,
// contents, and system-specific data.
// The ID is automatically generated and is unique for each call to this
// function.
func NewWriteRequest(keyname string, contents []byte, sysSpecific any) (Request, error) {
	var sysSpecificJSON json.RawMessage
	if sysSpecific != nil {
		b, err := json.Marshal(sysSpecific)
		if err != nil {
			return Request{}, err
		}
		sysSpecificJSON = b
	}
	return Request{
		ID:          NextID(),
		Keyname:     keyname,
		Write:       true,
		Contents:    base64.StdEncoding.EncodeToString(contents),
		SysSpecific: sysSpecificJSON,
	}, nil
}

// NewResponse creates a Response with the given contents, error, and system-specific data.
func (req Request) NewResponse(contents []byte, responseError *Error, sysSpecific any) (Response, error) {
	var sysSpecificJSON json.RawMessage
	if sysSpecific != nil {
		b, err := json.Marshal(sysSpecific)
		if err != nil {
			return Response{}, err
		}
		sysSpecificJSON = b
	}
	return Response{
		ID:          req.ID,
		Contents:    base64.StdEncoding.EncodeToString(contents),
		Error:       responseError,
		SysSpecific: sysSpecificJSON,
	}, nil
}

func (resp Response) UnmarshalSysSpecific(v any) error {
	if resp.SysSpecific == nil {
		return nil
	}
	return json.Unmarshal(resp.SysSpecific, v)
}

func (resp Response) UnmarshalContents() ([]byte, error) {
	return base64.StdEncoding.DecodeString(resp.Contents)
}
