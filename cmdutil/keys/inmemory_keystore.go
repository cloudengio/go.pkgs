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
	"io/fs"
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
	keys map[KeySpec]Info
}

// NewInMemoryKeyStore creates a new InMemoryKeyStore instance.
func NewInMemoryKeyStore() *InMemoryKeyStore {
	return &InMemoryKeyStore{
		keys: make(map[KeySpec]Info),
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
		ims.unmarshalList(asList)
		return nil
	}
	var asMap map[string]keyInfo
	err = node.Decode(&asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	ims.unmarshalMap(asMap)
	return nil
}

func (ims *InMemoryKeyStore) unmarshalList(asList []keyInfo) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	if ims.keys == nil {
		ims.keys = make(map[KeySpec]Info)
	}
	for _, ki := range asList {
		info := copyInfo(ki)
		ims.keys[KeySpec{User: info.User, ID: info.ID}] = info
	}
}

func (ims *InMemoryKeyStore) unmarshalMap(asMap map[string]keyInfo) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	if ims.keys == nil {
		ims.keys = make(map[KeySpec]Info)
	}
	for k, info := range asMap {
		info.ID = k
		ki := copyInfo(info)
		ims.keys[KeySpec{User: ki.User, ID: ki.ID}] = ki
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface to allow
// unmarshaling from both a list and a map of keys.
// The unmarshaled keys update any existing keys in the store, using User+ID as the key.
// textutil.TrimUnicodeQuotes is used on the ID, User, and Token fields.
func (ims *InMemoryKeyStore) UnmarshalJSON(data []byte) error {
	var asList []keyInfo
	err := json.Unmarshal(data, &asList)
	if err == nil {
		ims.unmarshalList(asList)
		return nil
	}
	var asMap map[string]keyInfo
	err = json.Unmarshal(data, &asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	ims.unmarshalMap(asMap)
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

// KeySpecs returns the owners of keys in the store, sorted by ID and User.
func (ims *InMemoryKeyStore) KeySpecs() []KeySpec {
	keys := ims.getSortedKeys()
	owners := make([]KeySpec, len(keys))
	for i, key := range keys {
		owners[i] = KeySpec{User: key.User, ID: key.ID}
	}
	return owners
}

// Add adds a key to the store. If a key with the same user and ID already
// exists, it will be overwritten.
func (ims *InMemoryKeyStore) Add(key Info) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	if ims.keys == nil {
		ims.keys = make(map[KeySpec]Info)
	}
	ims.keys[KeySpec{User: key.User, ID: key.ID}] = key
}

// Get retrieves a key by its user and ID. It returns the key and a boolean
// indicating whether the key was found. If user is not specified it will search
// for a unique key by ID alone. If there are multiple keys with the same id
// Get will return false and it is left to the caller to use GetOwned with
// a user specified.
func (ims *InMemoryKeyStore) Get(user, id string) (Info, bool) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	if len(user) == 0 {
		return ims.getUniqueLocked(id)
	}
	return ims.getOwnedLocked(user, id)
}

// getOwnedLocked is the exact-match lookup shared by Get and GetOwned. It
// assumes ims.mu is already held (for reading) by the caller and must not
// acquire it itself: RLock is not reentrant in the presence of a concurrent,
// blocked Lock call (see sync.RWMutex), so a second, nested RLock call from
// the same goroutine that already holds one can deadlock against a writer
// that is waiting for that first RLock to be released.
func (ims *InMemoryKeyStore) getOwnedLocked(user, id string) (Info, bool) {
	ko := KeySpec{User: user, ID: id}
	if key, ok := ims.keys[ko]; ok {
		return key, true
	}
	return Info{}, false
}

// GetOwned retrieves a key by its exact user and ID, performing no fallback
// even when user is empty: by definition only one or zero keys can exist for
// any specific user (even the empty, "unowned", one) and ID. See Get for a
// lenient lookup that also matches by ID alone when the ID is unambiguous.
func (ims *InMemoryKeyStore) GetOwned(user, id string) (Info, bool) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return ims.getOwnedLocked(user, id)
}

// getUniqueWithStatusLocked returns the matched Info and the match count
// (0, 1, or 2 for multiple matches). It assumes ims.mu is already held by the caller.
func (ims *InMemoryKeyStore) getUniqueWithStatusLocked(id string) (Info, int) {
	if id == "" {
		return Info{}, 0
	}
	var found Info
	var count int
	for _, key := range ims.keys {
		if key.ID == id {
			found = key
			count++
			if count > 1 {
				return Info{}, 2
			}
		}
	}
	return found, count
}

// getUniqueLocked is the by-ID-alone lookup shared by Get and GetUnique. Like
// getOwnedLocked, it assumes ims.mu is already held by the caller and must
// not acquire it itself. An empty id never matches, not even a key that was
// itself given an empty id.
func (ims *InMemoryKeyStore) getUniqueLocked(id string) (Info, bool) {
	ki, count := ims.getUniqueWithStatusLocked(id)
	return ki, count == 1
}

// GetSpecs retrieves keys for the provided specs under a single read lock,
// ensuring an atomic view across all requested keys. If any key cannot be
// found or is ambiguous, or if any spec has an empty ID, an error is returned.
func (ims *InMemoryKeyStore) GetSpecs(specs ...KeySpec) ([]Info, error) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()

	infos := make([]Info, 0, len(specs))
	for _, spec := range specs {
		if len(spec.ID) == 0 {
			clear(infos)
			if len(spec.User) == 0 {
				return nil, fmt.Errorf("empty key id")
			}
			return nil, fmt.Errorf("empty key id for user %q", spec.User)
		}
		if len(spec.User) == 0 {
			ki, count := ims.getUniqueWithStatusLocked(spec.ID)
			switch count {
			case 1:
				infos = append(infos, ki)
			case 0:
				clear(infos)
				return nil, fmt.Errorf("key %q not found", spec.ID)
			default:
				clear(infos)
				return nil, fmt.Errorf("ambiguous key %q: multiple users match this id", spec.ID)
			}
			continue
		}
		ki, ok := ims.getOwnedLocked(spec.User, spec.ID)
		if !ok {
			clear(infos)
			return nil, fmt.Errorf("key not found for user %q and id %q", spec.User, spec.ID)
		}
		infos = append(infos, ki)
	}
	return infos, nil
}

// GetUnique retrieves a key by its ID only if it is unique across all users.
// It returns the key and a boolean indicating whether a unique key was found.
func (ims *InMemoryKeyStore) GetUnique(id string) (Info, bool) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	return ims.getUniqueLocked(id)
}

// Delete removes a key from the store by its user and ID.
func (ims *InMemoryKeyStore) Delete(user, id string) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	delete(ims.keys, KeySpec{User: user, ID: id})
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

// WriteJSON marshals the InMemoryKeyStore to JSON and writes it to a file
// using the provided file.WriteFileFS.
func (ims *InMemoryKeyStore) WriteJSON(ctx context.Context, wfs file.WriteFileFS, name string, perm fs.FileMode) error {
	if len(name) == 0 {
		return fmt.Errorf("no keychain item name provided")
	}
	data, err := ims.MarshalJSON()
	if err != nil {
		return err
	}
	return wfs.WriteFileCtx(ctx, name, data, perm)
}

// WriteYAML marshals the InMemoryKeyStore to YAML and writes it to a file
// using the provided file.WriteFileFS.
func (ims *InMemoryKeyStore) WriteYAML(ctx context.Context, wfs file.WriteFileFS, name string, perm fs.FileMode) error {
	if len(name) == 0 {
		return fmt.Errorf("no keychain item name provided")
	}
	data, err := yaml.Marshal(ims)
	if err != nil {
		return err
	}
	return wfs.WriteFileCtx(ctx, name, data, perm)
}

func (ims *InMemoryKeyStore) Keys() []Info {
	return ims.getSortedKeys()
}
