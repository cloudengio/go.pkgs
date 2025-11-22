// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"cloudeng.io/security/keys/keychain/plugins"
)

func TestNewRequest(t *testing.T) {
	type sysSpec struct {
		Field string `json:"field"`
	}

	// Test case 1: Basic request with sysSpecific
	keyname := "test-key"
	spec := sysSpec{Field: "value"}
	req, err := plugins.NewRequest(keyname, spec)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	if req.Keyname != keyname {
		t.Errorf("got %q, want %q", req.Keyname, keyname)
	}
	if req.ID <= 0 {
		t.Errorf("got %d, want > 0", req.ID)
	}

	var gotSpec sysSpec
	if err := json.Unmarshal(req.SysSpecific, &gotSpec); err != nil {
		t.Fatalf("failed to unmarshal sysSpecific: %v", err)
	}
	if gotSpec != spec {
		t.Errorf("got %v, want %v", gotSpec, spec)
	}

	// Test case 2: Request with nil sysSpecific
	req2, err := plugins.NewRequest("key2", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req2.SysSpecific != nil {
		t.Errorf("got %v, want nil", req2.SysSpecific)
	}
	if req2.ID <= req.ID {
		t.Errorf("ID did not increment: got %d, previous %d", req2.ID, req.ID)
	}

	// Test case 3: JSON marshal error
	// Channels cannot be marshaled
	_, err = plugins.NewRequest("key3", make(chan int))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewResponse(t *testing.T) {
	req := plugins.Request{
		ID:      123,
		Keyname: "test-key",
	}

	contents := []byte("secret-data")
	respErr := &plugins.Error{
		Message: "something went wrong",
		Detail:  "error details",
	}
	type sysSpec struct {
		Info string `json:"info"`
	}
	spec := sysSpec{Info: "meta"}

	// Test case 1: Response with error and contents
	resp := req.NewResponse(contents, respErr)
	err := resp.WithSysSpecific(spec)
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}

	if resp.ID != req.ID {
		t.Errorf("got %d, want %d", resp.ID, req.ID)
	}

	decoded := resp.Contents
	if !bytes.Equal(decoded, contents) {
		t.Errorf("got %q, want %q", string(decoded), string(contents))
	}

	if resp.Error == nil {
		t.Fatal("expected error, got nil")
	}

	if resp.Error != nil && (resp.Error.Error() != respErr.Error()) {
		t.Errorf("got %q, want %q", resp.Error, respErr.Error())
	}

	var gotSpec sysSpec
	if err := json.Unmarshal(resp.SysSpecific, &gotSpec); err != nil {
		t.Fatalf("failed to unmarshal sysSpecific: %v", err)
	}
	if gotSpec != spec {
		t.Errorf("got %v, want %v", gotSpec, spec)
	}

	// Test case 2: Response with nil error and nil sysSpecific
	resp2 := req.NewResponse(contents, nil)
	if resp2.Error != nil {
		t.Errorf("got %q, want nil", resp2.Error)
	}
	if resp2.SysSpecific != nil {
		t.Errorf("got %v, want nil", resp2.SysSpecific)
	}

	// Test case 3: JSON marshal error
	err = req.NewResponse(contents, nil).WithSysSpecific(make(chan int))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestErrorNotFound(t *testing.T) {
	notFoundErr := plugins.NewErrorKeyNotFound("my-key")
	req, err := plugins.NewRequest("a key", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	resp := req.NewResponse(nil, notFoundErr)
	if !errors.Is(resp.Error, plugins.ErrKeyNotFound) {
		t.Errorf("expected error to be ErrKeyNotFound, got %v", resp.Error)
	}
}

func TestErrorKeyExists(t *testing.T) {
	keyExistsErr := plugins.NewErrorKeyExists("my-key")
	req, err := plugins.NewRequest("a key", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	resp := req.NewResponse(nil, keyExistsErr)
	if !errors.Is(resp.Error, plugins.ErrKeyExists) {
		t.Errorf("expected error to be ErrKeyExists, got %v", resp.Error)
	}
}

func TestNewWriteRequest(t *testing.T) {
	type sysSpec struct {
		Overwrite bool `json:"overwrite"`
	}

	keyname := "new-key"
	contents := []byte("new-secret")
	spec := sysSpec{Overwrite: true}

	// Test case 1: Basic write request
	req, err := plugins.NewWriteRequest(keyname, contents, spec)
	if err != nil {
		t.Fatalf("NewWriteRequest failed: %v", err)
	}

	if req.Keyname != keyname {
		t.Errorf("got keyname %q, want %q", req.Keyname, keyname)
	}
	if req.ID <= 0 {
		t.Errorf("got ID %d, want > 0", req.ID)
	}

	decoded := req.Contents
	if !bytes.Equal(decoded, contents) {
		t.Errorf("got contents %q, want %q", string(decoded), string(contents))
	}

	var gotSpec sysSpec
	if err := json.Unmarshal(req.SysSpecific, &gotSpec); err != nil {
		t.Fatalf("failed to unmarshal sysSpecific: %v", err)
	}
	if gotSpec != spec {
		t.Errorf("got spec %v, want %v", gotSpec, spec)
	}

	// Test case 2: nil sysSpecific
	req2, err := plugins.NewWriteRequest("key2", contents, nil)
	if err != nil {
		t.Fatalf("NewWriteRequest failed: %v", err)
	}
	if req2.SysSpecific != nil {
		t.Errorf("got spec %v, want nil", req2.SysSpecific)
	}

	// Test case 3: JSON marshal error
	_, err = plugins.NewWriteRequest("key3", contents, make(chan int))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
