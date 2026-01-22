// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package keys provides types and utilities for managing API keys/tokens.
// A key consists of an identifier, an optional user, a token value,
// and optional extra information. The package includes an in-memory key store
// for storing and retrieving keys, as well as context utilities for passing
// key stores across API boundaries.
package keys

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"cloudeng.io/file"
	"gopkg.in/yaml.v3"
)

// InMemoryKeyStore is a simple in-memory key store intended for
// passing a small number of keys within an application. It will
// typically be stored in a context.Context to ease passing it across
// API boundaries.
type InMemoryKeyStore struct {
	mu   sync.RWMutex
	keys []Info
}

// NewInMemoryKeyStore creates a new InMemoryKeyStore instance.
func NewInMemoryKeyStore() *InMemoryKeyStore {
	return &InMemoryKeyStore{}
}

func copyInfoList(src []keyInfo) []Info {
	dest := make([]Info, len(src))
	for i, ki := range src {
		dest[i] = copyInfo(ki)
	}
	return dest
}

// UnmarshalYAML implements the yaml.Unmarshaler interface to allow
// unmarshaling from both a list and a map of keys.
// textutil.TrimUnicodeQuotes is used on the ID, User, and Token fields.
func (ims *InMemoryKeyStore) UnmarshalYAML(node *yaml.Node) error {
	var asList []keyInfo
	err := node.Decode(&asList)
	if err == nil {
		ims.keys = copyInfoList(asList)
		return nil
	}
	var asMap map[string]keyInfo
	err = node.Decode(&asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	for id, info := range asMap {
		info.ID = id
		ims.keys = append(ims.keys, copyInfo(info))
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface to allow
// unmarshaling from both a list and a map of keys.
func (ims *InMemoryKeyStore) UnmarshalJSON(data []byte) error {
	var asList []keyInfo
	err := json.Unmarshal(data, &asList)
	if err == nil {
		ims.keys = copyInfoList(asList)
		return nil
	}
	var asMap map[string]keyInfo
	err = json.Unmarshal(data, &asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	for id, info := range asMap {
		info.ID = id
		ims.keys = append(ims.keys, copyInfo(info))
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface to allow
// marshaling the InMemoryKeyStore to JSON.
func (ims *InMemoryKeyStore) MarshalJSON() ([]byte, error) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return json.Marshal(ims.keys)
}

// MarshalYAML implements the yaml.Marshaler interface to allow
// marshaling the InMemoryKeyStore to YAML.
func (ims *InMemoryKeyStore) MarshalYAML() (any, error) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return ims.keys, nil
}

// KeyOwners returns the owners of keys in the store.
func (ims *InMemoryKeyStore) KeyOwners() []KeyOwner {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	owners := make([]KeyOwner, len(ims.keys))
	for i, key := range ims.keys {
		owners[i] = KeyOwner{ID: key.ID, User: key.User}
	}
	return owners
}

func (ims *InMemoryKeyStore) Add(key Info) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	ims.keys = append(ims.keys, key)
}

// Get retrieves a key by its ID. It returns the key and a boolean
// indicating whether the key was found.
func (ims *InMemoryKeyStore) Get(id string) (Info, bool) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	for _, key := range ims.keys {
		if key.ID == id {
			return key, true
		}
	}
	return Info{}, false
}

func (ims *InMemoryKeyStore) Len() int {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return len(ims.keys)
}

// ReadJSON reads key information from a JSON file using the provided
// file.ReadFileFS and unmarshals it into the InMemoryKeyStore.
func (ims *InMemoryKeyStore) ReadJSON(ctx context.Context, fs file.ReadFileFS, name string) error {
	if len(name) == 0 {
		return fmt.Errorf("no keychain item name provided")
	}
	data, err := fs.ReadFileCtx(ctx, name)
	if err != nil {
		return err
	}
	return ims.UnmarshalJSON(data)
}

// ReadYAML reads key information from a YAML file using the provided
// file.ReadFileFS and unmarshals it into the InMemoryKeyStore.
func (ims *InMemoryKeyStore) ReadYAML(ctx context.Context, fs file.ReadFileFS, name string) error {
	if len(name) == 0 {
		return fmt.Errorf("no keychain item name provided")
	}
	data, err := fs.ReadFileCtx(ctx, name)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, ims)
}
