// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"gopkg.in/yaml.v3"
)

// KeyOwner represents the owner of a key, identified by an ID and an optional user.
type KeyOwner struct {
	ID   string
	User string
}

func (ko KeyOwner) String() string {
	if ko.User != "" {
		return ko.ID + "[" + ko.User + "]"
	}
	return ko.ID
}

// Token represents an API token. It is intended for temporary use
// with the Clear() method being called to zero the token value when
// it is no longer needed, typically using a defer statement.
// It consists of an ID and a token value with the ID purely for
// identification purposes.
type Token struct {
	KeyOwner
	token []byte
}

// Value returns the value of the token.
func (t Token) Value() []byte {
	return t.token
}

// Clear zeros the token value.
func (t *Token) Clear() {
	t.KeyOwner = KeyOwner{}
	for i := range t.token {
		t.token[i] = 0
	}
}

func (t Token) String() string {
	return t.KeyOwner.String() + ":****"
}

// NewToken creates a new Token instance, cloning the provided value
// and zeroing the input slice.
func NewToken(id, user string, value []byte) Token {
	t := Token{KeyOwner: KeyOwner{ID: id, User: user}, token: slices.Clone(value)}
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

// NewInfo creates a new Info instance with the specified id, user, token.
// The token slice is cloned and the input slice is zeroed. Extra information
// can be set using WithExtra and accessed using UnmarshalExtra.
func NewInfo(id, user string, token []byte) Info {
	i := Info{
		ID:    id,
		User:  user,
		token: slices.Clone(token),
	}
	for i := range token {
		token[i] = 0
	}
	return i
}

// WithExtra sets the extra information for the key. Extra information can
// be accessed using UnmarshalExtra or GetExtra.
func (k *Info) WithExtra(v any) {
	k.extraAny = v
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
	return &Token{KeyOwner: KeyOwner{ID: k.ID, User: k.User}, token: slices.Clone(k.token)}
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

func (k *Info) UnmarshalJSON(data []byte) error {
	var kv keyInfo
	if err := json.Unmarshal(data, &kv); err != nil {
		return err
	}
	k.ID = kv.ID
	k.User = kv.User
	k.token = []byte(kv.Token)
	k.extraJSON = kv.ExtraJSON
	return nil
}

func (k *Info) UnmarshalYAML(node *yaml.Node) error {
	var kv keyInfo
	if err := node.Decode(&kv); err != nil {
		return err
	}
	k.ID = kv.ID
	k.User = kv.User
	k.token = []byte(kv.Token)
	k.extraYAML = kv.ExtraYAML
	return nil
}

func (k Info) MarshalJSON() ([]byte, error) {
	kv := keyInfo{
		ID:    k.ID,
		User:  k.User,
		Token: string(k.token),
	}
	var err error
	switch {
	case k.extraAny != nil:
		kv.ExtraJSON, err = json.Marshal(k.extraAny)
		if err != nil {
			return nil, err
		}
	case k.extraJSON != nil:
		kv.ExtraJSON = k.extraJSON
	case k.extraYAML.Kind != 0:
		var ka any
		if err := k.extraYAML.Decode(&ka); err != nil {
			return nil, err
		}
		kv.ExtraJSON, err = json.Marshal(ka)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(kv)
}

type keyInfoYAMLAny struct {
	ID    string `yaml:"key_id"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
	Extra any    `yaml:"extra,omitempty"`
}

func (k Info) MarshalYAML() (any, error) {
	kv := keyInfoYAMLAny{
		ID:    k.ID,
		User:  k.User,
		Token: string(k.token),
	}
	switch {
	case k.extraAny != nil:
		kv.Extra = k.extraAny
	case k.extraJSON != nil:
		if err := json.Unmarshal(k.extraJSON, &kv.Extra); err != nil {
			return nil, err
		}
	case k.extraYAML.Kind != 0:
		// ExtraYAML to be a yaml.Node and not any, otherwise the
		// yaml package will panic.
		return keyInfo{
			ID:        k.ID,
			User:      k.User,
			Token:     string(k.token),
			ExtraYAML: k.extraYAML,
		}, nil
	}
	return kv, nil
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

func (k Info) handleExtra(v any) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return false
	}
	et := reflect.TypeOf(k.extraAny)
	if et.AssignableTo(rv.Type().Elem()) {
		rv.Elem().Set(reflect.ValueOf(k.extraAny))
		return true
	}
	return false
}

// UnmarshalExtra unmarshals the extra json, yaml, or explicitly stored extra
// information into the provided value. It does not modify the stored extra information.
func (k Info) UnmarshalExtra(v any) error {
	if k.extraJSON == nil && k.extraYAML.Kind == 0 {
		if k.extraAny == nil {
			return fmt.Errorf("no extra unmarshalled information for key_id: %v", k.ID)
		}
		if k.handleExtra(v) {
			return nil
		}
		buf, err := json.Marshal(k.extraAny)
		if err != nil {
			return fmt.Errorf("failed to marshal extra json for key_id: %v: %w", k.ID, err)
		}
		return json.Unmarshal(buf, v)
	}
	if k.extraJSON != nil {
		return k.extraFromJSON(v)
	}
	return k.extraFromYAML(v)
}

// GetExtra returns the extra information for the key.
func (k Info) GetExtra() any {
	return k.extraAny
}
