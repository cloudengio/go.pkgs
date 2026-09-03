// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keychaintestutil_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"slices"
	"testing"

	"cloudeng.io/encoding/json/jsonmsgs"
	"cloudeng.io/file"
	"cloudeng.io/security/keys/keychain/keychaintestutil"
	"cloudeng.io/security/keys/keychain/plugins"
)

func TestPluginMemoryStore(t *testing.T) {
	p := keychaintestutil.New()

	// Initial get on empty store
	if _, ok := p.Get("key1"); ok {
		t.Errorf("expected key1 to not exist")
	}

	// Set and Get
	p.Set("key1", []byte("val1"))
	p.Set("key2", []byte("val2"))

	got, ok := p.Get("key1")
	if !ok || string(got) != "val1" {
		t.Errorf("Get(key1) = %q, %v, want %q, true", got, ok, "val1")
	}

	// Verify mutating returned slice does not affect internal store
	got[0] = 'X'
	got2, _ := p.Get("key1")
	if string(got2) != "val1" {
		t.Errorf("internal store mutated via returned slice")
	}

	// Keys list
	keysList := p.Keys()
	slices.Sort(keysList)
	if !slices.Equal(keysList, []string{"key1", "key2"}) {
		t.Errorf("Keys() = %v, want [key1, key2]", keysList)
	}

	// Delete
	p.Delete("key1")
	if _, ok := p.Get("key1"); ok {
		t.Errorf("expected key1 to be deleted")
	}

	// Clear
	p.Clear()
	if len(p.Keys()) != 0 {
		t.Errorf("expected empty store after Clear(), got %d keys", len(p.Keys()))
	}
}

func TestPluginHandleRequest(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()

	type customMetadata struct {
		Account string `json:"account"`
	}
	meta := customMetadata{Account: "acc-1"}

	// 1. Read non-existent key
	readReq, err := plugins.NewRequest("missing-key", meta)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp := p.HandleRequest(ctx, readReq)
	if resp.ID != readReq.ID {
		t.Errorf("resp.ID = %d, want %d", resp.ID, readReq.ID)
	}
	if !errors.Is(resp.Error, plugins.ErrKeyNotFound) {
		t.Errorf("resp.Error = %v, want ErrKeyNotFound", resp.Error)
	}
	var gotMeta customMetadata
	if err := resp.UnmarshalPluginSpecific(&gotMeta); err != nil {
		t.Fatalf("UnmarshalPluginSpecific: %v", err)
	}
	if gotMeta != meta {
		t.Errorf("gotMeta = %+v, want %+v", gotMeta, meta)
	}

	// 2. Write key
	writeReq, err := plugins.NewWriteRequest("k1", []byte("secret123"), meta)
	if err != nil {
		t.Fatalf("NewWriteRequest: %v", err)
	}
	writeResp := p.HandleRequest(ctx, writeReq)
	if writeResp.ID != writeReq.ID {
		t.Errorf("writeResp.ID = %d, want %d", writeResp.ID, writeReq.ID)
	}
	if writeResp.Error != nil {
		t.Errorf("writeResp.Error = %v, want nil", writeResp.Error)
	}

	// 3. Read back written key
	readReq2, _ := plugins.NewRequest("k1", nil)
	readResp2 := p.HandleRequest(ctx, readReq2)
	if readResp2.Error != nil {
		t.Errorf("readResp2.Error = %v, want nil", readResp2.Error)
	}
	if string(readResp2.Contents) != "secret123" {
		t.Errorf("readResp2.Contents = %q, want %q", readResp2.Contents, "secret123")
	}

}

func TestPluginHandleRequestInjectedErrors(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	p.Set("k1", []byte("secret123"))

	readReq2, err := plugins.NewRequest("k1", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// 4. Injected key error
	customErr := &plugins.Error{Message: "custom error", Detail: "custom detail"}
	p.SetError("k1", customErr)
	errResp := p.HandleRequest(ctx, readReq2)
	if errResp.Error == nil || errResp.Error.Message != "custom error" {
		t.Errorf("errResp.Error = %v, want custom error", errResp.Error)
	}

	// Clear injected error for k1
	p.SetError("k1", nil)
	okResp := p.HandleRequest(ctx, readReq2)
	if okResp.Error != nil {
		t.Errorf("okResp.Error = %v, want nil after clearing error", okResp.Error)
	}

	// 5. Injected default error
	defaultErr := &plugins.Error{Message: "default error", Detail: "all fail"}
	p.SetDefaultError(defaultErr)
	defResp := p.HandleRequest(ctx, readReq2)
	if defResp.Error == nil || defResp.Error.Message != "default error" {
		t.Errorf("defResp.Error = %v, want default error", defResp.Error)
	}
}

func TestPluginServeIO(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	p.Set("secret-key", []byte("payload"))

	// Valid Request
	req, _ := plugins.NewRequest("secret-key", nil)
	var inBuf bytes.Buffer
	if err := plugins.WriteRequest(jsonmsgs.NewMessager(&inBuf, nil), req); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var outBuf bytes.Buffer
	if err := p.ServeIO(ctx, &inBuf, &outBuf); err != nil {
		t.Fatalf("ServeIO: %v", err)
	}

	resp, err := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(&outBuf)))
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.ID != req.ID || string(resp.Contents) != "payload" || resp.Error != nil {
		t.Errorf("unexpected response: %+v", resp)
	}

	// Malformed JSON input
	inBuf.Reset()
	outBuf.Reset()
	inBuf.WriteString("invalid-json")
	if err := p.ServeIO(ctx, &inBuf, &outBuf); err != nil {
		t.Fatalf("ServeIO on invalid json: %v", err)
	}
	errResp, err := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(&outBuf)))
	if err != nil {
		t.Fatalf("read error response: %v", err)
	}
	if errResp.Error == nil || errResp.Error.Message != "failed to decode request" {
		t.Errorf("errResp.Error = %+v, want decode error", errResp.Error)
	}
}

func TestFS(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()

	rwFS := keychaintestutil.NewFS(p, true)
	if rwFS.Plugin() != p {
		t.Errorf("rwFS.Plugin() != p")
	}

	// Write file
	if err := rwFS.WriteFileCtx(ctx, "test-key", []byte("fs-data"), 0600); err != nil {
		t.Fatalf("WriteFileCtx: %v", err)
	}
	if err := rwFS.WriteFile("test-key2", []byte("fs-data2"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Read file
	data1, err := rwFS.ReadFileCtx(ctx, "test-key")
	if err != nil {
		t.Fatalf("ReadFileCtx: %v", err)
	}
	if string(data1) != "fs-data" {
		t.Errorf("ReadFileCtx = %q, want %q", data1, "fs-data")
	}

	data2, err := rwFS.ReadFile("test-key2")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data2) != "fs-data2" {
		t.Errorf("ReadFile = %q, want %q", data2, "fs-data2")
	}

	// Read non-existent
	if _, err := rwFS.ReadFileCtx(ctx, "nonexistent"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}

	// Read-only FS
	roFS := keychaintestutil.NewFS(p, false)
	if err := roFS.WriteFileCtx(ctx, "any", []byte("data"), 0600); !errors.Is(err, plugins.ErrReadOnly) {
		t.Errorf("expected ErrReadOnly from WriteFileCtx, got %v", err)
	}
	if err := roFS.WriteFile("any", []byte("data"), 0600); !errors.Is(err, plugins.ErrReadOnly) {
		t.Errorf("expected ErrReadOnly from WriteFile, got %v", err)
	}
}

func TestContextInjection(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	p.Set("injected-key", []byte("injected-secret"))

	rwFS := keychaintestutil.NewFS(p, true)
	ctxWithFS := file.ContextWithReadWriteFS(ctx, rwFS)

	retrievedFS, ok := file.ReadWriteFSFromContext(ctxWithFS)
	if !ok || retrievedFS != rwFS {
		t.Fatalf("ReadWriteFSFromContext failed")
	}

	data, err := retrievedFS.ReadFileCtx(ctxWithFS, "injected-key")
	if err != nil {
		t.Fatalf("ReadFileCtx: %v", err)
	}
	if string(data) != "injected-secret" {
		t.Errorf("got %q, want injected-secret", data)
	}
}

func TestServer(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	p.Set("server-key", []byte("server-value"))

	server, err := keychaintestutil.StartServer(ctx, p)
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	defer server.Close()

	if server.Plugin() != p {
		t.Errorf("server.Plugin() != p")
	}
	if server.Address() == "" {
		t.Errorf("server.Address() is empty")
	}

	// Test Run CLI forwarding to server
	req, _ := plugins.NewRequest("server-key", nil)
	var inBuf bytes.Buffer
	_ = plugins.WriteRequest(jsonmsgs.NewMessager(&inBuf, nil), req)

	var outBuf bytes.Buffer
	var stderr bytes.Buffer
	if err := keychaintestutil.Run(ctx, &inBuf, &outBuf, &stderr, "--socket="+server.Address()); err != nil {
		t.Fatalf("Run with --socket: %v", err)
	}

	resp, err := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(&outBuf)))
	if err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if string(resp.Contents) != "server-value" {
		t.Errorf("resp.Contents = %q, want server-value", resp.Contents)
	}
}

func TestRunFlags(t *testing.T) {
	ctx := context.Background()

	// 1. Error flag
	req, _ := plugins.NewRequest("k", nil)
	var inBuf bytes.Buffer
	_ = plugins.WriteRequest(jsonmsgs.NewMessager(&inBuf, nil), req)

	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	if err := keychaintestutil.Run(ctx, &inBuf, &outBuf, &errBuf, "--error=forced failure"); err != nil {
		t.Fatalf("Run --error: %v", err)
	}
	resp, _ := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(&outBuf)))
	if resp.Error == nil || resp.Error.Detail != "forced failure" {
		t.Errorf("resp.Error = %+v, want forced failure", resp.Error)
	}

	// 2. Static contents flag matching keyname
	inBuf.Reset()
	outBuf.Reset()
	_ = plugins.WriteRequest(jsonmsgs.NewMessager(&inBuf, nil), req)
	if err := keychaintestutil.Run(ctx, &inBuf, &outBuf, &errBuf, "--contents=static-secret", "--keyname=k"); err != nil {
		t.Fatalf("Run --contents: %v", err)
	}
	resp2, _ := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(&outBuf)))
	if string(resp2.Contents) != "static-secret" {
		t.Errorf("resp2.Contents = %q, want static-secret", resp2.Contents)
	}

	// 3. Static contents flag with mismatched keyname
	inBuf.Reset()
	outBuf.Reset()
	_ = plugins.WriteRequest(jsonmsgs.NewMessager(&inBuf, nil), req)
	if err := keychaintestutil.Run(ctx, &inBuf, &outBuf, &errBuf, "--contents=static-secret", "--keyname=other-key"); err != nil {
		t.Fatalf("Run --contents mismatched: %v", err)
	}
	resp3, _ := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(&outBuf)))
	if !errors.Is(resp3.Error, plugins.ErrKeyNotFound) {
		t.Errorf("resp3.Error = %+v, want ErrKeyNotFound", resp3.Error)
	}
}

// Building the plugin binary is expensive, so both phases of the integration
// test share the one binary and server built here.
func TestBuildPluginBinaryAndRunExtPlugin(t *testing.T) {
	ctx := context.Background()
	binPath := keychaintestutil.BuildPluginBinary(t)

	p := keychaintestutil.New()
	server, err := keychaintestutil.StartServer(ctx, p)
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	defer server.Close()

	// Set env var so subprocess plugin connects to the shared in-memory server
	t.Setenv(keychaintestutil.SocketEnvVar, server.Address())

	checkRunExtPlugin(ctx, t, binPath, p)
	checkPluginFS(ctx, t, binPath)
}

// checkRunExtPlugin writes a key and reads it back, each in its own subprocess
// plugin run, and verifies the write landed in the shared in-memory store.
func checkRunExtPlugin(ctx context.Context, t *testing.T, binPath string, p *keychaintestutil.Plugin) {
	t.Helper()

	// 1. Write via RunExtPlugin
	writeReq, err := plugins.NewWriteRequest("integration-key", []byte("shared-memory-value"), nil)
	if err != nil {
		t.Fatalf("NewWriteRequest: %v", err)
	}
	writeResp, err := plugins.RunExtPlugin(ctx, binPath, writeReq)
	if err != nil {
		t.Fatalf("RunExtPlugin write: %v", err)
	}
	if writeResp.Error != nil {
		t.Fatalf("writeResp.Error: %v", writeResp.Error)
	}

	// Verify directly in in-memory store
	val, ok := p.Get("integration-key")
	if !ok || string(val) != "shared-memory-value" {
		t.Fatalf("p.Get = %q, %v, want shared-memory-value, true", val, ok)
	}

	// 2. Read back via RunExtPlugin in another subprocess
	readReq, err := plugins.NewRequest("integration-key", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	readResp, err := plugins.RunExtPlugin(ctx, binPath, readReq)
	if err != nil {
		t.Fatalf("RunExtPlugin read: %v", err)
	}
	if readResp.Error != nil {
		t.Fatalf("readResp.Error: %v", readResp.Error)
	}
	if string(readResp.Contents) != "shared-memory-value" {
		t.Errorf("readResp.Contents = %q, want shared-memory-value", readResp.Contents)
	}
}

// checkPluginFS exercises the file.FS wrapper around the plugin binary, reading
// the key written by checkRunExtPlugin and round-tripping one of its own.
func checkPluginFS(ctx context.Context, t *testing.T, binPath string) {
	t.Helper()

	// 3. Test with plugins.NewFS (wrapping the binary)
	pluginFS := plugins.NewFS(binPath, true, nil)
	fsReadVal, err := pluginFS.ReadFileCtx(ctx, "integration-key")
	if err != nil {
		t.Fatalf("pluginFS.ReadFileCtx: %v", err)
	}
	if string(fsReadVal) != "shared-memory-value" {
		t.Errorf("fsReadVal = %q, want shared-memory-value", fsReadVal)
	}

	if err := pluginFS.WriteFileCtx(ctx, "key-from-fs", []byte("fs-secret"), fs.FileMode(0600)); err != nil {
		t.Fatalf("pluginFS.WriteFileCtx: %v", err)
	}
	fsReadVal2, err := pluginFS.ReadFileCtx(ctx, "key-from-fs")
	if err != nil {
		t.Fatalf("pluginFS.ReadFileCtx: %v", err)
	}
	if string(fsReadVal2) != "fs-secret" {
		t.Errorf("fsReadVal2 = %q, want fs-secret", fsReadVal2)
	}
}

func TestPluginUnsupportedVersion(t *testing.T) {
	ctx := context.Background()
	p := keychaintestutil.New()
	p.Set("key1", []byte("val1"))

	req := plugins.Request{
		Version: plugins.RequestCurrentVersion + 1,
		ID:      plugins.NextID(),
		Keyname: "key1",
	}
	resp := p.HandleRequest(ctx, req)
	if resp.Error == nil {
		t.Fatal("expected an error for an unsupported request version, got nil")
	}
	if !errors.Is(resp.Error, plugins.ErrUnsupportedVersion) {
		t.Errorf("expected error to be ErrUnsupportedVersion, got %v", resp.Error)
	}
	if len(resp.Contents) != 0 {
		t.Errorf("expected no contents, got %q", resp.Contents)
	}

	// A supported version (including zero for older clients) is handled.
	for _, v := range []int32{0, plugins.RequestVersion1} {
		req := plugins.Request{Version: v, ID: plugins.NextID(), Keyname: "key1"}
		resp := p.HandleRequest(ctx, req)
		if resp.Error != nil {
			t.Errorf("version %d: unexpected error: %v", v, resp.Error)
		}
		if got, want := string(resp.Contents), "val1"; got != want {
			t.Errorf("version %d: got %q, want %q", v, got, want)
		}
	}
}

func TestRunUnsupportedVersion(t *testing.T) {
	ctx := context.Background()
	req := plugins.Request{
		Version: plugins.RequestCurrentVersion + 1,
		ID:      plugins.NextID(),
		Keyname: "key1",
	}
	var inBuf bytes.Buffer
	if err := plugins.WriteRequest(jsonmsgs.NewMessager(&inBuf, nil), req); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	if err := keychaintestutil.Run(ctx, &inBuf, out, os.Stderr, "--contents=static"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	resp, err := plugins.ReadResponse(jsonmsgs.NewMessager(nil, io.NopCloser(out)))
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected an error for an unsupported request version, got nil")
	}
	if !errors.Is(resp.Error, plugins.ErrUnsupportedVersion) {
		t.Errorf("expected error to be ErrUnsupportedVersion, got %v", resp.Error)
	}
}
