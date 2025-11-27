// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"encoding/json"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"gopkg.in/yaml.v3"
)

const (
	yamlList = `
- key_id: key1
  token: value1
  user: user1
- key_id: key2
  token: value2
  user: user2
`
	yamlMap = `
key1:
  token: value1
  user: user1
key2:
  token: value2
  user: user2
`
	jsonList = `[
    {"key_id": "key1", "token": "value1", "user": "user1"},
    {"key_id": "key2", "token": "value2", "user": "user2"}
]`
	jsonMap = `{
    "key1": {"token": "value1", "user": "user1"},
    "key2": {"token": "value2", "user": "user2"}
}`
)

func TestUnmarshalYAML(t *testing.T) {
	var ks keys.InmemoryKeyStore
	if err := yaml.Unmarshal([]byte(yamlList), &ks); err != nil {
		t.Fatalf("yaml list: %v", err)
	}
	verifyKeys(t, &ks)

	ks = keys.InmemoryKeyStore{}
	if err := yaml.Unmarshal([]byte(yamlMap), &ks); err != nil {
		t.Fatalf("yaml map: %v", err)
	}
	verifyKeys(t, &ks)
}

func TestUnmarshalJSON(t *testing.T) {
	var ks keys.InmemoryKeyStore
	if err := json.Unmarshal([]byte(jsonList), &ks); err != nil {
		t.Fatalf("json list: %v", err)
	}
	verifyKeys(t, &ks)

	ks = keys.InmemoryKeyStore{}
	if err := json.Unmarshal([]byte(jsonMap), &ks); err != nil {
		t.Fatalf("json map: %v", err)
	}
	verifyKeys(t, &ks)
}

func verifyKeys(t *testing.T, ks *keys.InmemoryKeyStore) {
	k1, ok := ks.Get("key1")
	if !ok {
		t.Fatalf("key1 not found")
	}
	if got, want := string(k1.Token().Value()), "value1"; got != want {
		t.Errorf("key1: got %v, want %v", got, want)
	}
	if got, want := k1.User, "user1"; got != want {
		t.Errorf("key1 user: got %v, want %v", got, want)
	}

	k2, ok := ks.Get("key2")
	if !ok {
		t.Fatalf("key2 not found")
	}
	if got, want := string(k2.Token().Value()), "value2"; got != want {
		t.Errorf("key2: got %v, want %v", got, want)
	}
	if got, want := k2.User, "user2"; got != want {
		t.Errorf("key2 user: got %v, want %v", got, want)
	}
}

func TestContextFunctions(t *testing.T) {
	store := keys.NewInmemoryKeyStore()
	k1 := &keys.Info{ID: "ctx-key", User: "ctx-user"}
	k1.SetToken(keys.NewToken([]byte("ctx-token")))
	store.Add(*k1)

	ctx := keys.ContextWithKeyStore(context.Background(), store)

	// Test successful retrieval
	retrievedKey, ok := keys.KeyInfoFromContextForID(ctx, "ctx-key")
	if !ok {
		t.Fatal("expected to find key in context")
	}
	if got, want := retrievedKey.ID, k1.ID; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := retrievedKey.User, k1.User; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := string(retrievedKey.Token().Value()), string(k1.Token().Value()); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Test retrieval of non-existent key from context
	_, ok = keys.KeyInfoFromContextForID(ctx, "non-existent")
	if ok {
		t.Error("did not expect to find non-existent key in context")
	}

	// Test retrieval from a context without the store
	emptyCtx := context.Background()
	_, ok = keys.KeyInfoFromContextForID(emptyCtx, "ctx-key")
	if ok {
		t.Error("did not expect to find key in empty context")
	}
}

type extraType struct {
	Scope string `json:"scope" yaml:"scope"`
}

const (
	yamlListExtra = `
- key_id: key1
  token: value1
  user: user1
  extra:
    scope: read
- key_id: key2
  token: value2
  user: user2
  extra:
    scope: write
`
	yamlMapExtra = `
key1:
  token: value1
  user: user1
  extra:
    scope: read
key2:
  token: value2
  user: user2
  extra:
    scope: write
`
	jsonListExtra = `[
    {"key_id": "key1", "token": "value1", "user": "user1", "extra": {"scope": "read"}},
    {"key_id": "key2", "token": "value2", "user": "user2", "extra": {"scope": "write"}}
]`
	jsonMapExtra = `{
    "key1": {"token": "value1", "user": "user1", "extra": {"scope": "read"}},
    "key2": {"token": "value2", "user": "user2", "extra": {"scope": "write"}}
}`
)

func TestUnmarshalYAMLExtra(t *testing.T) {
	var ks keys.InmemoryKeyStore
	if err := yaml.Unmarshal([]byte(yamlListExtra), &ks); err != nil {
		t.Fatalf("yaml list: %v", err)
	}
	verifyKeysExtra(t, &ks)

	ks = keys.InmemoryKeyStore{}
	if err := yaml.Unmarshal([]byte(yamlMapExtra), &ks); err != nil {
		t.Fatalf("yaml map: %v", err)
	}
	verifyKeysExtra(t, &ks)
}

func TestUnmarshalJSONExtra(t *testing.T) {
	var ks keys.InmemoryKeyStore
	if err := json.Unmarshal([]byte(jsonListExtra), &ks); err != nil {
		t.Fatalf("json list: %v", err)
	}
	verifyKeysExtra(t, &ks)

	ks = keys.InmemoryKeyStore{}
	if err := json.Unmarshal([]byte(jsonMapExtra), &ks); err != nil {
		t.Fatalf("json map: %v", err)
	}
	verifyKeysExtra(t, &ks)
}

func verifyKeysExtra(t *testing.T, ks *keys.InmemoryKeyStore) {
	k1, ok := ks.Get("key1")
	if !ok {
		t.Fatalf("key1 not found")
	}
	var e1 extraType
	if err := k1.ExtraAs(&e1); err != nil {
		t.Fatalf("key1 extra: %v", err)
	}
	if got, want := e1.Scope, "read"; got != want {
		t.Errorf("key1 scope: got %v, want %v", got, want)
	}

	k2, ok := ks.Get("key2")
	if !ok {
		t.Fatalf("key2 not found")
	}
	var e2 extraType
	if err := k2.ExtraAs(&e2); err != nil {
		t.Fatalf("key2 extra: %v", err)
	}
	if got, want := e2.Scope, "write"; got != want {
		t.Errorf("key2 scope: got %v, want %v", got, want)
	}
}

func TestToken(t *testing.T) {
	val := []byte("secret")
	tok := keys.NewToken(val)
	if got, want := string(tok.Value()), "secret"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tok.String(), "****"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// Verify input was cleared
	if string(val) == "secret" {
		t.Errorf("input slice was not cleared")
	}
	tok.Clear()
	for _, b := range tok.Value() {
		if b != 0 {
			t.Errorf("token was not cleared")
		}
	}
}

func TestInfo(t *testing.T) {
	val := []byte("secret")
	extra := map[string]string{"a": "b"}
	info := keys.NewInfo("id", "user", val, extra)

	if got, want := info.ID, "id"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := info.User, "user"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := string(info.Token().Value()), "secret"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := info.String(), "id[user]"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := info.Extra().(map[string]string)["a"], "b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Verify input was cleared
	if string(val) == "secret" {
		t.Errorf("input slice was not cleared")
	}

	newToken := keys.NewToken([]byte("new-secret"))
	info.SetToken(newToken)
	if got, want := string(info.Token().Value()), "new-secret"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMoreContextFunctions(t *testing.T) {
	ctx := context.Background()

	// ContextWithoutKeyStore
	store := keys.NewInmemoryKeyStore()
	ctxWithStore := keys.ContextWithKeyStore(ctx, store)
	if _, ok := keys.KeyStoreFromContext(ctxWithStore); !ok {
		t.Fatal("expected store in context")
	}

	ctxNoStore := keys.ContextWithoutKeyStore(ctxWithStore)
	if _, ok := keys.KeyStoreFromContext(ctxNoStore); ok {
		t.Fatal("expected no store in context")
	}

	// ContextWithKey
	k1 := keys.NewInfo("k1", "u1", []byte("t1"), nil)
	ctxWithKey := keys.ContextWithKey(ctx, k1)

	// Should have created a store and added the key
	storeFromCtx, ok := keys.KeyStoreFromContext(ctxWithKey)
	if !ok {
		t.Fatal("expected store to be created")
	}

	gotKey, ok := storeFromCtx.Get("k1")
	if !ok {
		t.Fatal("expected key to be in store")
	}
	if gotKey.ID != "k1" {
		t.Errorf("got %v, want k1", gotKey.ID)
	}

	// Add another key to existing store
	k2 := keys.NewInfo("k2", "u2", []byte("t2"), nil)
	ctxWithKey2 := keys.ContextWithKey(ctxWithKey, k2)

	storeFromCtx2, ok := keys.KeyStoreFromContext(ctxWithKey2)
	if !ok {
		t.Fatal("expected store")
	}
	if _, ok := storeFromCtx2.Get("k2"); !ok {
		t.Fatal("expected k2")
	}
	if _, ok := storeFromCtx2.Get("k1"); !ok {
		t.Fatal("expected k1 to still be there")
	}
}
