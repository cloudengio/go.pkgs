// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package plugins defines the request/response protocol used to communicate
// with an out-of-process keychain plugin, and the client side of that
// protocol (see FS and RunExtPlugin).
//
// Requests are versioned so that a client built against a newer version of
// this package can detect a plugin binary that is too old to understand its
// requests; this matters because the client and the plugin are separate
// binaries that are frequently built and installed independently of each
// other. Requests created by NewRequest and NewWriteRequest carry
// RequestCurrentVersion. A plugin must call Request.CheckVersion on every
// request it decodes and, if a non-nil error is returned, send that error
// back as the response's Error rather than attempting to service the
// request. Responses are not versioned: a plugin only ever replies to a
// request whose version it has accepted.
package plugins

import (
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

// NewErrorKeyExists creates a new Error indicating that the specified key
// already exists that is compatible with errors.Is and ErrorKeyExists.
func NewErrorKeyExists(keyname string) *Error {
	return &Error{
		Message: "key already exists",
		Detail:  keyname,
	}
}

// NewErrorUnsupportedVersion creates a new Error indicating that the
// request's version is newer than this implementation supports, that is
// compatible with errors.Is and ErrUnsupportedVersion.
func NewErrorUnsupportedVersion(version int32) *Error {
	return &Error{
		Message: "unsupported request version",
		Detail:  fmt.Sprintf("request version %d is newer than the latest supported version %d", version, RequestCurrentVersion),
	}
}

// ErrUnsupportedVersion can be used as the target of errors.Is to check for
// an unsupported request version error.
var ErrUnsupportedVersion = NewErrorUnsupportedVersion(0)

// ErrKeyNotFound can be used as the target of errors.Is to check for a
// key not found error.
var ErrKeyNotFound = NewErrorKeyNotFound("")

// ErrKeyExists can be used as the target of errors.Is to check for a
// key already exists error.
var ErrKeyExists = NewErrorKeyExists("")

// Error represents an error returned by a plugin.
type Error struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Stderr  string `json:"-"` // Stderr is the stder output from the plugin and is
	// filled in by the client of the plugin.
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.Detail)
}

func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}
	if err, ok := errors.AsType[*Error](target); ok {
		return e.Message == err.Message
	}
	return false
}

// Request represents the request to the keychain plugin.
type Request struct {
	Version        int32           `json:"version,omitempty"`
	ID             int32           `json:"id,omitempty"`
	Keyname        string          `json:"keyname"`
	Write          bool            `json:"write,omitempty"`
	Contents       []byte          `json:"contents,omitempty"`
	PluginSpecific json.RawMessage `json:"plugin_specific,omitempty"`
}

const (
	// RequestVersion1 is the initial version of the request format.
	RequestVersion1 int32 = 1

	// RequestCurrentVersion is the version of the request format used by
	// requests created by this package. A plugin built against this package
	// must accept requests at this version or lower and reject requests with
	// a higher version, see Request.CheckVersion.
	RequestCurrentVersion = RequestVersion1
)

// CheckVersion verifies that the request's version can be handled by this
// implementation of the plugin protocol. Requests with a version at or below
// RequestCurrentVersion are accepted (including a zero version from clients
// that predate versioning) and nil is returned. Requests with a newer version
// return an error compatible with ErrUnsupportedVersion that plugins should
// send back to the client as the response's Error.
func (req Request) CheckVersion() *Error {
	if req.Version > RequestCurrentVersion {
		return NewErrorUnsupportedVersion(req.Version)
	}
	return nil
}

// Response represents the response from the keychain plugin.
type Response struct {
	ID             int32           `json:"id,omitempty"`
	Contents       []byte          `json:"contents,omitempty"`
	Stderr         string          `json:"-"` // Stderr is the stder output from the plugin and is filled in by RunExtPlugin.
	Error          *Error          `json:"error,omitempty"`
	PluginSpecific json.RawMessage `json:"plugin_specific,omitempty"`
}

var nextID int32 = 1

func NextID() int32 {
	return atomic.AddInt32(&nextID, 1)
}

// NewRequest creates a Request to read a key with the given keyname and
// system-specific data.
// The ID is automatically generated and is unique for each call to this
// function.
func NewRequest(keyname string, pluginSpecific any) (Request, error) {
	var pluginSpecificJSON json.RawMessage
	if pluginSpecific != nil {
		b, err := json.Marshal(pluginSpecific)
		if err != nil {
			return Request{}, err
		}
		pluginSpecificJSON = b
	}
	return Request{
		Version:        RequestVersion1,
		ID:             NextID(),
		Keyname:        keyname,
		PluginSpecific: pluginSpecificJSON,
	}, nil
}

// NewWriteRequest creates a Request to write a key with the given keyname,
// contents, and plugin-specific data.
// The ID is automatically generated and is unique for each call to this
// function.
func NewWriteRequest(keyname string, contents []byte, pluginSpecific any) (Request, error) {
	var pluginSpecificJSON json.RawMessage
	if pluginSpecific != nil {
		b, err := json.Marshal(pluginSpecific)
		if err != nil {
			return Request{}, err
		}
		pluginSpecificJSON = b
	}
	return Request{
		Version:        RequestVersion1,
		ID:             NextID(),
		Keyname:        keyname,
		Write:          true,
		Contents:       contents,
		PluginSpecific: pluginSpecificJSON,
	}, nil
}

// NewResponse creates a Response with the given contents and error.
func (req Request) NewResponse(contents []byte, responseError *Error) *Response {
	return &Response{
		ID:       req.ID,
		Contents: contents,
		Error:    responseError,
	}
}

// WithPluginSpecific sets the PluginSpecific field of the Response to the JSON
// encoding of the given pluginSpecific data.
func (resp *Response) WithPluginSpecific(pluginSpecific any) error {
	if pluginSpecific != nil {
		b, err := json.Marshal(pluginSpecific)
		if err != nil {
			return err
		}
		resp.PluginSpecific = b
	}
	return nil
}

func (resp Response) UnmarshalPluginSpecific(v any) error {
	if resp.PluginSpecific == nil {
		return nil
	}
	return json.Unmarshal(resp.PluginSpecific, v)
}

// AsError attempts to convert the given error to a *Error and returns it.
// If the error is not a *Error, it returns nil.
func AsError(err error) *Error {
	if e, ok := errors.AsType[*Error](err); ok {
		return e
	}
	return nil
}
