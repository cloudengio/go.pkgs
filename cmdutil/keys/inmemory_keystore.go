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
	"sort"
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
	keys map[KeyOwner]Info
}

// NewInMemoryKeyStore creates a new InMemoryKeyStore instance.
func NewInMemoryKeyStore() *InMemoryKeyStore {
	return &InMemoryKeyStore{
		keys: make(map[KeyOwner]Info),
	}
}

// UnmarshalYAML implements the yaml.Unmarshaler interface to allow
// unmarshaling from both a list and a map of keys.
// The unmarshaled keys update any existing keys in the store, using User+ID as the key.
// textutil.TrimUnicodeQuotes is used on the ID, User, and Token fields.
func (ims *InMemoryKeyStore) UnmarshalYAML(node *yaml.Node) error {
	var asList []keyInfo
	err := node.Decode(&asList)
	if err == nil {
		ims.mu.Lock()
		defer ims.mu.Unlock()
		if ims.keys == nil {
			ims.keys = make(map[KeyOwner]Info)
		}
		for _, ki := range asList {
			info := copyInfo(ki)
			ims.keys[KeyOwner{ID: info.ID, User: info.User}] = info
		}
		return nil
	}
	var asMap map[string]keyInfo
	err = node.Decode(&asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	ims.mu.Lock()
	defer ims.mu.Unlock()
	if ims.keys == nil {
		ims.keys = make(map[KeyOwner]Info)
	}
	for k, info := range asMap {
		info.ID = k
		ki := copyInfo(info)
		ims.keys[KeyOwner{ID: ki.ID, User: ki.User}] = ki
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface to allow
// unmarshaling from both a list and a map of keys.
// The unmarshaled keys update any existing keys in the store, using User+ID as the key.
// textutil.TrimUnicodeQuotes is used on the ID, User, and Token fields.
func (ims *InMemoryKeyStore) UnmarshalJSON(data []byte) error {
	var asList []keyInfo
	err := json.Unmarshal(data, &asList)
	if err == nil {
		ims.mu.Lock()
		defer ims.mu.Unlock()
		if ims.keys == nil {
			ims.keys = make(map[KeyOwner]Info)
		}
		for _, ki := range asList {
			info := copyInfo(ki)
			ims.keys[KeyOwner{ID: info.ID, User: info.User}] = info
		}
		return nil
	}
	var asMap map[string]keyInfo
	err = json.Unmarshal(data, &asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	ims.mu.Lock()
	defer ims.mu.Unlock()
	if ims.keys == nil {
		ims.keys = make(map[KeyOwner]Info)
	}
	for k, info := range asMap {
		info.ID = k
		ki := copyInfo(info)
		ims.keys[KeyOwner{ID: ki.ID, User: ki.User}] = ki
	}
	return nil
}

// getSortedKeys returns a deterministic list of Info objects sorted by ID and User.
func (ims *InMemoryKeyStore) getSortedKeys() []Info {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	vals := make([]Info, 0, len(ims.keys))
	for _, v := range ims.keys {
		vals = append(vals, v)
	}
	sort.Slice(vals, func(i, j int) bool {
		if vals[i].ID == vals[j].ID {
			return vals[i].User < vals[j].User
		}
		return vals[i].ID < vals[j].ID
	})
	return vals
}

// MarshalJSON implements the json.Marshaler interface to allow
// marshaling the InMemoryKeyStore to JSON.
func (ims *InMemoryKeyStore) MarshalJSON() ([]byte, error) {
	return json.Marshal(ims.getSortedKeys())
}

// MarshalYAML implements the yaml.Marshaler interface to allow
// marshaling the InMemoryKeyStore to YAML.
func (ims *InMemoryKeyStore) MarshalYAML() (any, error) {
	return ims.getSortedKeys(), nil
}

// KeyOwners returns the owners of keys in the store, sorted by ID and User.
func (ims *InMemoryKeyStore) KeyOwners() []KeyOwner {
	keys := ims.getSortedKeys()
	owners := make([]KeyOwner, len(keys))
	for i, key := range keys {
		owners[i] = KeyOwner{ID: key.ID, User: key.User}
	}
	return owners
}

func (ims *InMemoryKeyStore) Add(key Info) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	if ims.keys == nil {
		ims.keys = make(map[KeyOwner]Info)
	}
	ims.keys[KeyOwner{ID: key.ID, User: key.User}] = key
}

// Get retrieves a key by its user and ID. It returns the key and a boolean
// indicating whether the key was found.
func (ims *InMemoryKeyStore) Get(user, id string) (Info, bool) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	for _, key := range ims.keys {
		if key.User == user && key.ID == id {
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
