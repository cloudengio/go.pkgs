// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys

import (
	"context"
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"
)

// KeyInfo represents a specific key and associated information and is intended
// to be reused and referred to by it's key_id.
type KeyInfo struct {
	ID    string `yaml:"key_id"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
	Extra any    `yaml:"extra,omitempty"` // Extra can be used to store additional information about the key
}

func (k KeyInfo) String() string {
	return k.ID + "[" + k.User + "]	"
}

// ExtraAs attempts to unmarshal the Extra field into the provided struct.
func (k KeyInfo) ExtraAs(v any) error {
	if k.Extra == nil {
		return fmt.Errorf("no extra information available for key_id: %v", k.ID)
	}
	extraBytes, err := yaml.Marshal(k.Extra)
	if err != nil {
		return fmt.Errorf("failed to marshal extra information for key_id: %v: %w", k.ID, err)
	}
	return yaml.Unmarshal(extraBytes, v)
}

// InmemoryKeyStore is a simple in-memory key store intended for
// passing a small number of keys within an application. It will
// typically be stored in a context.Context to ease passing it across
// API boundaries.
type InmemoryKeyStore struct {
	keys []KeyInfo
}

// NewInmemoryKeyStore creates a new InmemoryKeyStore instance.
func NewInmemoryKeyStore() *InmemoryKeyStore {
	return &InmemoryKeyStore{}
}

// UnmarshalYAML implements the yaml.Unmarshaler interface to allow
// unmarshaling from both a list and a map of keys.
func (ims *InmemoryKeyStore) UnmarshalYAML(node *yaml.Node) error {
	var asList []KeyInfo
	err := node.Decode(&asList)
	if err == nil {
		ims.keys = asList
		return nil
	}
	var asMap map[string]KeyInfo
	err = node.Decode(&asMap)
	if err != nil {
		return fmt.Errorf("failed to decode input as either a list or a map of keys: %w", err)
	}
	for id, info := range asMap {
		info.ID = id
		ims.keys = append(ims.keys, info)
	}
	return nil
}

func (s *InmemoryKeyStore) AddKey(key KeyInfo) {
	s.keys = append(s.keys, key)
}

// GetKey retrieves a key by its ID. It returns the key and a boolean
// indicating whether the key was found.
func (s *InmemoryKeyStore) GetKey(id string) (KeyInfo, bool) {
	for _, key := range s.keys {
		if key.ID == id {
			return key, true
		}
	}
	return KeyInfo{}, false
}

// GetAllKeys returns all keys in the store.
func (s *InmemoryKeyStore) GetAllKeys() []KeyInfo {
	return slices.Clone(s.keys)
}

type ctxKey struct{}

func ContextWithAuth(ctx context.Context, ims InmemoryKeyStore) context.Context {
	return context.WithValue(ctx, ctxKey{}, ims)
}

func AuthFromContextForID(ctx context.Context, id string) (KeyInfo, bool) {
	am, ok := ctx.Value(ctxKey{}).(InmemoryKeyStore)
	if !ok {
		return KeyInfo{}, false
	}
	return am.GetKey(id)
}
