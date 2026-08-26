// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonerr provides support for working with errors sent over the wire
// using JSON. It provides a registry for recreating local instances of concrete
// error types that have been received from a remote process. The type of
// an error is represented by its package name and type name (e.g. "example.com/pkg.MyError").
package jsonerr

import (
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type errorFactory func(jsontext.Value) (error, error)

var (
	errorTypesMu sync.RWMutex
	errorTypes   = map[string]errorFactory{}
)

func getErrorFactory(typeName string) (errorFactory, bool) {
	errorTypesMu.RLock()
	defer errorTypesMu.RUnlock()
	f, ok := errorTypes[typeName]
	return f, ok
}

// TypeNameForError returns the fully qualified type name of err
// (e.g. "example.com/pkg.MyError"). Returns "" for nil.
func TypeNameForError(err error) string {
	if err == nil {
		return ""
	}
	typ := reflect.TypeOf(err)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.PkgPath() + "." + typ.Name()
}

func RegisterType[T any, PT interface {
	*T
	error
}]() {
	typeName := TypeNameForError(PT(new(T)))
	errorTypesMu.Lock()
	defer errorTypesMu.Unlock()
	errorTypes[typeName] = func(raw jsontext.Value) (error, error) {
		v := PT(new(T))
		if err := json.Unmarshal(raw, v); err != nil {
			return nil, fmt.Errorf("decode %s: %w", typeName, err)
		}
		return v, nil
	}
}

// MarshalError marshals an error into an Error struct suitable for
// transmission over the wire. TypeNameForError(err) is used to set Error.Type,
// Error.Error is set to err.Error(), and Error.Detail is set to the JSON-encoded
// representation of err. Error.Type must be registered using RegisterType
// by the recipient of the marshaled error in order to unmarshal the error back
// into its corresponding concrete type.
func MarshalError(err error) ([]byte, error) {
	if err == nil {
		return json.Marshal(Error{})
	}
	typeName := TypeNameForError(err)
	detail, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal error detail: %w", marshalErr)
	}
	je := Error{
		Error:  err.Error(),
		Type:   typeName,
		Detail: jsontext.Value(detail),
	}
	return json.Marshal(je)
}

var defaultUnmarshalError = NewUnmarshalError(DefaultUnknownTypeHandler)

// UnmarshalError expects data to be a JSON-encoded Error struct. It uses the Type
// field to determine the concrete type of the error, and unmarshals the Detail field
// into that type. If the Type is not registered, it returns an error using
// DefaultUnknownTypeHandler.
func UnmarshalError(data []byte) (error, error) {
	return defaultUnmarshalError.Unmarshal(data)
}

// DefaultUnknownTypeHandler uses errors.New(err.Error) to create an error.
// The Type and Detail fields are ignored.
func DefaultUnknownTypeHandler(err Error) error {
	return errors.New(err.Error)
}

// UnknownTypeHandler is a function that handles errors of unknown types.
type UnknownTypeHandler func(err Error) error

// UnmarshalErrorWithHandler implements json.UnmarshalerFrom using a custom
// UnknownTypeHandler for unknown error types. The unmarshaled error is stored
// in the Err field.
type UnmarshalErrorWithHandler struct {
	Err     error
	handler UnknownTypeHandler
}

// NewUnmarshalError creates a new UnmarshalErrorWithHandler with the given UnknownTypeHandler.
// If handler is nil, DefaultUnknownTypeHandler is used.
func NewUnmarshalError(handler UnknownTypeHandler) *UnmarshalErrorWithHandler {
	if handler == nil {
		handler = DefaultUnknownTypeHandler
	}
	return &UnmarshalErrorWithHandler{handler: handler}
}

// Unmarshal decodes data as a JSON-encoded Error and returns the concrete Go
// error. The first return value is the decoded application error (or the
// handler's result for unknown types); the second is any decoding failure.
func (ue *UnmarshalErrorWithHandler) Unmarshal(data []byte) (error, error) {
	if string(data) == "null" || len(data) == 0 {
		return nil, nil
	}
	var env Error
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	if env.Type == "" && env.Error == "" && (len(env.Detail) == 0 || string(env.Detail) == "null") {
		return nil, nil
	}
	factory, ok := getErrorFactory(env.Type)
	if !ok {
		if ue.handler != nil {
			return ue.handler(env), nil
		}
		return fmt.Errorf("unknown remote error type %q", env.Type), nil
	}
	return factory(env.Detail)
}

func (ue *UnmarshalErrorWithHandler) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var env Error
	if err := env.UnmarshalJSONFrom(dec); err != nil {
		return err
	}
	if env.Type == "" && env.Error == "" && (len(env.Detail) == 0 || string(env.Detail) == "null") {
		ue.Err = nil
		return nil
	}
	factory, ok := getErrorFactory(env.Type)
	if !ok {
		if ue.handler != nil {
			ue.Err = ue.handler(env)
			return nil
		}
		return fmt.Errorf("unknown remote error type %q", env.Type)
	}
	appErr, decErr := factory(env.Detail)
	if decErr != nil {
		return decErr
	}
	ue.Err = appErr
	return nil
}

// Error represents the 'on-the-wire' error representation. All errors
// must be converted to this form before being sent over the wire, and converted
// back to an error on the receiving side. The Type field is used to determine
// the concrete type of the error on the receiving side, and the Detail field
// contains the JSON-encoded representation of the error.
type Error struct {
	Error  string         `json:"error"`
	Type   string         `json:"type"`
	Detail jsontext.Value `json:"detail"`
}

func (e *Error) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, kv := range [2][2]string{
		{"error", e.Error},
		{"type", e.Type},
	} {
		if err := enc.WriteToken(jsontext.String(kv[0])); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(kv[1])); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.String("detail")); err != nil {
		return err
	}
	if len(e.Detail) > 0 {
		if err := enc.WriteValue(e.Detail); err != nil {
			return err
		}
	} else {
		if err := enc.WriteToken(jsontext.Null); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

func (e *Error) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if tok, err := dec.ReadToken(); err != nil {
		return err
	} else if tok.Kind() != '{' {
		return fmt.Errorf("expected '{', got %v", tok.Kind())
	}
	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		switch keyTok.String() {
		case "error":
			valTok, err := dec.ReadToken()
			if err != nil {
				return err
			}
			e.Error = valTok.String()
		case "type":
			valTok, err := dec.ReadToken()
			if err != nil {
				return err
			}
			e.Type = valTok.String()
		case "detail":
			val, err := dec.ReadValue()
			if err != nil {
				return err
			}
			e.Detail = val.Clone() // Clone because ReadValue returns a slice into the decoder's internal buffer
		default:
			if _, err := dec.ReadValue(); err != nil {
				return err
			}
		}
	}
	_, err := dec.ReadToken() // consume '}'
	return err
}
