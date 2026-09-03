// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package keychaintestutil provides test utilities for keychain plugins,
// including an in-memory implementation of the keychain plugin protocol.
package keychaintestutil

import (
	"context"
	"io"
	"sync"

	"cloudeng.io/encoding/json/jsonmsgs"
	"cloudeng.io/security/keys/keychain/plugins"
)

// Plugin is an in-memory implementation of a keychain plugin.
// It stores keys in memory and implements the plugin request/response protocol.
type Plugin struct {
	mu           sync.RWMutex
	items        map[string][]byte
	errors       map[string]*plugins.Error
	defaultError *plugins.Error
}

// New creates a new in-memory Plugin.
func New() *Plugin {
	return &Plugin{
		items:  make(map[string][]byte),
		errors: make(map[string]*plugins.Error),
	}
}

// Set stores a key and its contents in memory.
func (p *Plugin) Set(keyname string, contents []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[keyname] = append([]byte(nil), contents...)
}

// Get retrieves the contents of a key from memory.
func (p *Plugin) Get(keyname string) ([]byte, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	data, ok := p.items[keyname]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), data...), true
}

// Delete removes a key from memory.
func (p *Plugin) Delete(keyname string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, keyname)
}

// Clear removes all keys and configured errors from memory.
func (p *Plugin) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = make(map[string][]byte)
	p.errors = make(map[string]*plugins.Error)
	p.defaultError = nil
}

// Keys returns a slice of all key names currently in memory.
func (p *Plugin) Keys() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	keys := make([]string, 0, len(p.items))
	for k := range p.items {
		keys = append(keys, k)
	}
	return keys
}

// SetError configures a specific error to return for operations on the given keyname.
func (p *Plugin) SetError(keyname string, err *plugins.Error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err == nil {
		delete(p.errors, keyname)
	} else {
		p.errors[keyname] = err
	}
}

// SetDefaultError configures a default error to return for all requests.
func (p *Plugin) SetDefaultError(err *plugins.Error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultError = err
}

// HandleRequest processes a single plugins.Request and returns the corresponding plugins.Response.
// Requests with a version newer than plugins.RequestCurrentVersion are
// rejected with an error compatible with plugins.ErrUnsupportedVersion.
func (p *Plugin) HandleRequest(_ context.Context, req plugins.Request) plugins.Response {
	p.mu.Lock()
	defer p.mu.Unlock()

	if verr := req.CheckVersion(); verr != nil {
		resp := req.NewResponse(nil, verr)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return *resp
	}

	if p.defaultError != nil {
		resp := req.NewResponse(nil, p.defaultError)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return *resp
	}

	if err, ok := p.errors[req.Keyname]; ok {
		resp := req.NewResponse(nil, err)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return *resp
	}

	if req.Write {
		p.items[req.Keyname] = append([]byte(nil), req.Contents...)
		resp := req.NewResponse(nil, nil)
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return *resp
	}

	data, ok := p.items[req.Keyname]
	if !ok {
		resp := req.NewResponse(nil, plugins.NewErrorKeyNotFound(req.Keyname))
		_ = resp.WithPluginSpecific(req.PluginSpecific)
		return *resp
	}

	resp := req.NewResponse(append([]byte(nil), data...), nil)
	_ = resp.WithPluginSpecific(req.PluginSpecific)
	return *resp
}

// ServeIO reads a Request from r, handles it with HandleRequest,
// and writes the Response to w.
func (p *Plugin) ServeIO(ctx context.Context, r io.Reader, w io.Writer) error {
	msgr := jsonmsgs.NewMessager(w, io.NopCloser(r))
	req, err := plugins.ReadRequest(msgr)
	if err != nil {
		resp := plugins.Response{
			Error: &plugins.Error{
				Message: "failed to decode request",
				Detail:  err.Error(),
			},
		}
		return plugins.WriteResponse(msgr, resp)
	}
	resp := p.HandleRequest(ctx, req)
	return plugins.WriteResponse(msgr, resp)
}
