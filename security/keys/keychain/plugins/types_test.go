// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
	"cloudeng.io/security/keys/keychain/plugins"
)

func TestNewRequest(t *testing.T) {
	type sysSpec struct {
		Field string `json:"field"`
	}

	// Test case 1: Basic request with pluginSpecific
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
	if err := json.Unmarshal(req.PluginSpecific, &gotSpec); err != nil {
		t.Fatalf("failed to unmarshal pluginSpecific: %v", err)
	}
	if gotSpec != spec {
		t.Errorf("got %v, want %v", gotSpec, spec)
	}

	// Test case 2: Request with nil pluginSpecific
	req2, err := plugins.NewRequest("key2", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req2.PluginSpecific != nil {
		t.Errorf("got %v, want nil", req2.PluginSpecific)
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
	err := resp.WithPluginSpecific(spec)
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
	if err := json.Unmarshal(resp.PluginSpecific, &gotSpec); err != nil {
		t.Fatalf("failed to unmarshal pluginSpecific: %v", err)
	}
	if gotSpec != spec {
		t.Errorf("got %v, want %v", gotSpec, spec)
	}

	// Test case 2: Response with nil error and nil pluginSpecific
	resp2 := req.NewResponse(contents, nil)
	if resp2.Error != nil {
		t.Errorf("got %q, want nil", resp2.Error)
	}
	if resp2.PluginSpecific != nil {
		t.Errorf("got %v, want nil", resp2.PluginSpecific)
	}

	// Test case 3: JSON marshal error
	err = req.NewResponse(contents, nil).WithPluginSpecific(make(chan int))
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
	if err := json.Unmarshal(req.PluginSpecific, &gotSpec); err != nil {
		t.Fatalf("failed to unmarshal pluginSpecific: %v", err)
	}
	if gotSpec != spec {
		t.Errorf("got spec %v, want %v", gotSpec, spec)
	}

	// Test case 2: nil pluginSpecific
	req2, err := plugins.NewWriteRequest("key2", contents, nil)
	if err != nil {
		t.Fatalf("NewWriteRequest failed: %v", err)
	}
	if req2.PluginSpecific != nil {
		t.Errorf("got spec %v, want nil", req2.PluginSpecific)
	}

	// Test case 3: JSON marshal error
	_, err = plugins.NewWriteRequest("key3", contents, make(chan int))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequestVersion(t *testing.T) {
	req, err := plugins.NewRequest("key", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if got, want := req.Version, plugins.RequestVersion1; got != want {
		t.Errorf("NewRequest version: got %d, want %d", got, want)
	}
	wreq, err := plugins.NewWriteRequest("key", []byte("contents"), nil)
	if err != nil {
		t.Fatalf("NewWriteRequest failed: %v", err)
	}
	if got, want := wreq.Version, plugins.RequestVersion1; got != want {
		t.Errorf("NewWriteRequest version: got %d, want %d", got, want)
	}
}

func TestRequestCheckVersion(t *testing.T) {
	// Supported versions, including zero for clients that predate versioning.
	for _, v := range []int32{0, plugins.RequestVersion1, plugins.RequestCurrentVersion} {
		req := plugins.Request{Version: v}
		if verr := req.CheckVersion(); verr != nil {
			t.Errorf("version %d: unexpected error: %v", v, verr)
		}
	}

	req := plugins.Request{Version: plugins.RequestCurrentVersion + 1}
	verr := req.CheckVersion()
	if verr == nil {
		t.Fatal("expected an error for an unsupported version")
	}
	if !errors.Is(verr, plugins.ErrUnsupportedVersion) {
		t.Errorf("expected error to be ErrUnsupportedVersion, got %v", verr)
	}
	if errors.Is(verr, plugins.ErrKeyNotFound) {
		t.Errorf("unsupported version error should not match ErrKeyNotFound")
	}
}

func TestWriteReadRequest(t *testing.T) {
	type spec struct {
		Region string `json:"region"`
	}
	s := spec{Region: "us-west-2"}
	req, err := plugins.NewRequest("my-key", s)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	var buf bytes.Buffer
	msgr := jsonmsgs.NewMessager(io.NopCloser(&buf), &buf)
	if err := plugins.WriteRequest(msgr, req); err != nil {
		t.Fatalf("WriteRequest failed: %v", err)
	}

	gotReq, err := plugins.ReadRequest(msgr)
	if err != nil {
		t.Fatalf("ReadRequest failed: %v", err)
	}

	if gotReq.ID != req.ID || gotReq.Keyname != req.Keyname || gotReq.Version != req.Version {
		t.Errorf("got %+v, want %+v", gotReq, req)
	}

	var gotSpec spec
	if err := gotReq.UnmarshalPluginSpecific(&gotSpec); err != nil {
		t.Fatalf("UnmarshalPluginSpecific failed: %v", err)
	}
	if gotSpec != s {
		t.Errorf("got %v, want %v", gotSpec, s)
	}
}

func TestWriteReadResponse(t *testing.T) {
	type spec struct {
		Token string `json:"token"`
	}
	req, _ := plugins.NewRequest("key", nil)
	resp := req.NewResponse([]byte("secret-payload"), nil)
	if err := resp.WithPluginSpecific(spec{Token: "tok-123"}); err != nil {
		t.Fatalf("WithPluginSpecific failed: %v", err)
	}

	var buf bytes.Buffer
	msgr := jsonmsgs.NewMessager(io.NopCloser(&buf), &buf)
	if err := plugins.WriteResponse(msgr, *resp); err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}

	gotResp, err := plugins.ReadResponse(msgr)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}

	if gotResp.ID != resp.ID || string(gotResp.Contents) != string(resp.Contents) {
		t.Errorf("got %+v, want %+v", gotResp, resp)
	}

	var gotSpec spec
	if err := gotResp.UnmarshalPluginSpecific(&gotSpec); err != nil {
		t.Fatalf("UnmarshalPluginSpecific failed: %v", err)
	}
	if gotSpec.Token != "tok-123" {
		t.Errorf("got %v, want tok-123", gotSpec.Token)
	}
}

func TestMismatchedMessageType(t *testing.T) {
	// If sender writes a Response, but receiver calls ReadRequest,
	// jsonpayload.Decode should reject the mismatched type name.
	req, _ := plugins.NewRequest("key", nil)
	resp := req.NewResponse([]byte("data"), nil)

	var buf bytes.Buffer
	msgr := jsonmsgs.NewMessager(io.NopCloser(&buf), &buf)
	if err := plugins.WriteResponse(msgr, *resp); err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}

	_, err := plugins.ReadRequest(msgr)
	if err == nil {
		t.Fatal("expected error reading Response as Request, got nil")
	}
	if !strings.Contains(err.Error(), "expected type name") {
		t.Errorf("got error %v, want 'expected type name'", err)
	}
}
