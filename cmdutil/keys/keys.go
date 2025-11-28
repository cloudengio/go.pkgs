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
	"slices"
	"sync"

	"gopkg.in/yaml.v3"
)

// Token represents an API token. It is intended for temporary use
// with the Clear() method being called to zero the token value when
// it is no longer needed, typically using a defer statement.
type Token struct {
	token []byte
}

// Value returns the value of the token.
func (t Token) Value() []byte {
	return t.token
}

// Clear zeros the token value.
func (t *Token) Clear() {
	for i := range t.token {
		t.token[i] = 0
	}
}

func (t Token) String() string {
	return "****"
}

// NewToken creates a new Token instance, cloning the provided value
// and zeroing the input slice.
func NewToken(value []byte) Token {
	t := Token{token: slices.Clone(value)}
	for i := range value {
		value[i] = 0
	}
	return t
}

// Info represents a specific key and associated information and is intended
// to be reused and referred to by it's ID.
// It can be parsed from json or yaml representations with the following fields:
//   - key_id: the identifier for the key
//   - user: optional user associated with the key
//   - token: the token value
//   - extra: optional extra information as a json or yaml object
//
// An Info instance can be created/populated using NewInfo or by unmarshaling
// from json or yaml.
type Info struct {
	ID        string
	User      string
	token     []byte
	extraJSON json.RawMessage
	extraYAML yaml.Node
	extraAny  any
}

// NewInfo creates a new Info instance with the specified id, user, token, and
// extra information. The token slice is cloned and the input slice is zeroed.
func NewInfo(id, user string, token []byte, extra any) Info {
	i := Info{
		ID:       id,
		User:     user,
		token:    slices.Clone(token),
		extraAny: extra,
	}
	for i := range token {
		token[i] = 0
	}
	return i
}

type keyInfo struct {
	ID        string          `yaml:"key_id" json:"key_id"`
	User      string          `yaml:"user" json:"user"`
	Token     string          `yaml:"token" json:"token"`
	ExtraJSON json.RawMessage `yaml:"-" json:"extra,omitempty"`
	ExtraYAML yaml.Node       `yaml:"extra,omitempty" json:"-"`
}

// String returns a string representation of the KeyInfo with the Token
// and Extra fields redacted.
func (k Info) String() string {
	return k.ID + "[" + k.User + "]"
}

func (k Info) Token() *Token {
	return &Token{token: slices.Clone(k.token)}
}

func (k *Info) SetToken(t Token) {
	k.token = slices.Clone(t.token)
}

// Extra returns the extra information associated with the key. If no value
// was set using NewInfo, it will attempt to unmarshal the extra information
// from either the json or yaml representation.
func (k *Info) Extra() any {
	if k.extraAny != nil {
		return k.extraAny
	}
	if k.extraJSON != nil {
		var val any
		if json.Unmarshal(k.extraJSON, &val) == nil {
			k.extraAny = val
		}
	} else if k.extraYAML.Kind != 0 {
		var val any
		if k.extraYAML.Decode(&val) == nil {
			k.extraAny = val
		}
	}
	return k.extraAny
}

func (k Info) extraFromJSON(v any) error {
	if err := json.Unmarshal(k.extraJSON, v); err != nil {
		return fmt.Errorf("failed to unmarshal extra json for key_id: %v: %w", k.ID, err)
	}
	return nil
}

func (k Info) extraFromYAML(v any) error {
	if err := k.extraYAML.Decode(v); err != nil {
		return fmt.Errorf("failed to unmarshal extra yaml for key_id: %v: %w", k.ID, err)
	}
	return nil
}

// ExtraAs unmarshals the extra json or yaml information into the provided
// value. It does not modify the stored extra information.
func (k Info) ExtraAs(v any) error {
	if k.extraJSON == nil && k.extraYAML.Kind == 0 {
		return fmt.Errorf("no extra unmarshalled information for key_id: %v", k.ID)
	}
	if k.extraJSON != nil {
		return k.extraFromJSON(v)
	}
	return k.extraFromYAML(v)
}

// InmemoryKeyStore is a simple in-memory key store intended for
// passing a small number of keys within an application. It will
// typically be stored in a context.Context to ease passing it across
// API boundaries.
type InmemoryKeyStore struct {
	mu   sync.RWMutex
	keys []Info
}

// NewInmemoryKeyStore creates a new InmemoryKeyStore instance.
func NewInmemoryKeyStore() *InmemoryKeyStore {
	return &InmemoryKeyStore{}
}

func copyInfo(src keyInfo) Info {
	return Info{
		ID:        src.ID,
		User:      src.User,
		token:     []byte(src.Token),
		extraJSON: src.ExtraJSON,
		extraYAML: src.ExtraYAML,
	}
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
func (ims *InmemoryKeyStore) UnmarshalYAML(node *yaml.Node) error {
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
func (ims *InmemoryKeyStore) UnmarshalJSON(data []byte) error {
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

func (ims *InmemoryKeyStore) Add(key Info) {
	ims.mu.Lock()
	defer ims.mu.Unlock()
	ims.keys = append(ims.keys, key)
}

// Get retrieves a key by its ID. It returns the key and a boolean
// indicating whether the key was found.
func (ims *InmemoryKeyStore) Get(id string) (Info, bool) {
	ims.mu.RLock()
	defer ims.mu.RUnlock()
	for _, key := range ims.keys {
		if key.ID == id {
			return key, true
		}
	}
	return Info{}, false
}

type ctxKey struct{}

// ContextWithKeyStore returns a new context with the provided InmemoryKeyStore.
func ContextWithKeyStore(ctx context.Context, ims *InmemoryKeyStore) context.Context {
	return context.WithValue(ctx, ctxKey{}, ims)
}

// KeyStoreFromContext retrieves the InmemoryKeyStore from the context.
func KeyStoreFromContext(ctx context.Context) (*InmemoryKeyStore, bool) {
	am, ok := ctx.Value(ctxKey{}).(*InmemoryKeyStore)
	if !ok {
		return nil, false
	}
	return am, true
}

// ContextWithoutKeyStore returns a new context without an InmemoryKeyStore.
func ContextWithoutKeyStore(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, nil)
}

// KeyInfoFromContextForID retrieves the KeyInfo for the specified ID from the context.
func KeyInfoFromContextForID(ctx context.Context, id string) (Info, bool) {
	am, ok := ctx.Value(ctxKey{}).(*InmemoryKeyStore)
	if !ok {
		return Info{}, false
	}
	return am.Get(id)
}

func ContextWithKey(ctx context.Context, ki Info) context.Context {
	ims, ok := KeyStoreFromContext(ctx)
	if !ok {
		ims = NewInmemoryKeyStore()
		ctx = ContextWithKeyStore(ctx, ims)
	}
	ims.Add(ki)
	return ctx
}
