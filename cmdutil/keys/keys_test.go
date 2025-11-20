// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"reflect"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"gopkg.in/yaml.v3"
)

func TestInmemoryKeyStore(t *testing.T) {
	store := keys.NewInmemoryKeyStore()
	if store == nil {
		t.Fatal("NewInmemoryKeyStore returned nil")
	}

	if len(store.GetAllKeys()) != 0 {
		t.Error("new store should be empty")
	}

	k1 := keys.KeyInfo{ID: "key1", User: "user1", Token: "token1"}
	k2 := keys.KeyInfo{ID: "key2", User: "user2", Token: "token2"}

	store.AddKey(k1)
	store.AddKey(k2)

	// Test GetKey
	retrievedKey, ok := store.GetKey("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if !reflect.DeepEqual(retrievedKey, k1) {
		t.Errorf("got %v, want %v", retrievedKey, k1)
	}

	_, ok = store.GetKey("non-existent")
	if ok {
		t.Error("did not expect to find non-existent key")
	}

	// Test GetAllKeys
	allKeys := store.GetAllKeys()
	if len(allKeys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(allKeys))
	}
	if !reflect.DeepEqual(allKeys, []keys.KeyInfo{k1, k2}) {
		t.Errorf("got %v, want %v", allKeys, []keys.KeyInfo{k1, k2})
	}

	// Test that GetAllKeys returns a clone
	allKeys[0].ID = "modified"
	if store.GetAllKeys()[0].ID == "modified" {
		t.Error("GetAllKeys should return a clone, not a reference")
	}
}

func TestContextFunctions(t *testing.T) {
	store := keys.NewInmemoryKeyStore()
	k1 := keys.KeyInfo{ID: "ctx-key", User: "ctx-user", Token: "ctx-token"}
	store.AddKey(k1)

	ctx := keys.ContextWithAuth(context.Background(), *store)

	// Test successful retrieval
	retrievedKey, ok := keys.AuthFromContextForID(ctx, "ctx-key")
	if !ok {
		t.Fatal("expected to find key in context")
	}
	if !reflect.DeepEqual(retrievedKey, k1) {
		t.Errorf("got %v, want %v", retrievedKey, k1)
	}

	// Test retrieval of non-existent key from context
	_, ok = keys.AuthFromContextForID(ctx, "non-existent")
	if ok {
		t.Error("did not expect to find non-existent key in context")
	}

	// Test retrieval from a context without the store
	emptyCtx := context.Background()
	_, ok = keys.AuthFromContextForID(emptyCtx, "ctx-key")
	if ok {
		t.Error("did not expect to find key in empty context")
	}
}

type scopeType struct {
	Scope string `yaml:"scope"`
}

const (
	yamlDataArray = `
- key_id: key1
  user: user1
  token: token1
  extra:
    scope: read
- key_id: key2
  user: user2
  token: token2
`
	yamlDataMap = `
key1:
  user: user1
  token: token1
  extra:
    scope: read
key2:
  user: user2
  token: token2
`
)

func TestUnmarshalYAML(t *testing.T) {
	for _, yamlData := range []string{yamlDataArray, yamlDataMap} {
		store := keys.NewInmemoryKeyStore()
		err := yaml.Unmarshal([]byte(yamlData), store)
		if err != nil {
			t.Fatalf("UnmarshalYAML failed: %v", err)
		}

		allKeys := store.GetAllKeys()
		if len(allKeys) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(allKeys))
		}

		k1, ok := store.GetKey("key1")
		if !ok {
			t.Fatal("key1 not found")
		}
		if k1.User != "user1" || k1.Token != "token1" {
			t.Errorf("unexpected data for key1: %v", k1)
		}
		extra, ok := k1.Extra.(map[string]any)
		if !ok {
			t.Fatalf("unexpected type for extra data: %T", k1.Extra)
		}
		if extra["scope"] != "read" {
			t.Errorf("unexpected extra data: %v", extra)
		}

		var st scopeType
		if err := k1.ExtraAs(&st); err != nil {
			t.Errorf("failed to unmarshal extra data: %v", err)
		}
		if st.Scope != "read" {
			t.Errorf("unexpected scope value: %v", st.Scope)
		}
		if err := k1.ExtraAs(&scopeType{}); err != nil {
			t.Errorf("failed to unmarshal extra data: %v", err)
		}

		k2, ok := store.GetKey("key2")
		if !ok {
			t.Fatal("key2 not found")
		}
		if k2.User != "user2" || k2.Token != "token2" {
			t.Errorf("unexpected data for key2: %v", k2)
		}
	}
}

func TestUnmarshalYAMLInvalid(t *testing.T) {
	store := keys.NewInmemoryKeyStore()
	// Test invalid YAML
	invalidYAML := `not: valid: yaml`
	err := yaml.Unmarshal([]byte(invalidYAML), store)
	if err == nil {
		t.Fatal("expected an error for invalid YAML")
	}
}
