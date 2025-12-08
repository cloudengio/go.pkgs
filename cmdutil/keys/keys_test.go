// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"encoding/json"
	"strings"
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
	var ks keys.InMemoryKeyStore
	if err := yaml.Unmarshal([]byte(yamlList), &ks); err != nil {
		t.Fatalf("yaml list: %v", err)
	}
	verifyKeys(t, &ks)

	ks = keys.InMemoryKeyStore{}
	if err := yaml.Unmarshal([]byte(yamlMap), &ks); err != nil {
		t.Fatalf("yaml map: %v", err)
	}
	verifyKeys(t, &ks)
}

func TestUnmarshalJSON(t *testing.T) {
	var ks keys.InMemoryKeyStore
	if err := json.Unmarshal([]byte(jsonList), &ks); err != nil {
		t.Fatalf("json list: %v", err)
	}
	verifyKeys(t, &ks)

	ks = keys.InMemoryKeyStore{}
	if err := json.Unmarshal([]byte(jsonMap), &ks); err != nil {
		t.Fatalf("json map: %v", err)
	}
	verifyKeys(t, &ks)
}

func verifyKeys(t *testing.T, ks *keys.InMemoryKeyStore) {
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
	store := keys.NewInMemoryKeyStore()
	k1 := keys.NewInfo("ctx-key", "ctx-user", []byte("ctx-token"), nil)
	store.Add(k1)

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
	var ks keys.InMemoryKeyStore
	if err := yaml.Unmarshal([]byte(yamlListExtra), &ks); err != nil {
		t.Fatalf("yaml list: %v", err)
	}
	verifyKeysExtra(t, &ks)

	ks = keys.InMemoryKeyStore{}
	if err := yaml.Unmarshal([]byte(yamlMapExtra), &ks); err != nil {
		t.Fatalf("yaml map: %v", err)
	}
	verifyKeysExtra(t, &ks)
}

func TestUnmarshalJSONExtra(t *testing.T) {
	var ks keys.InMemoryKeyStore
	if err := json.Unmarshal([]byte(jsonListExtra), &ks); err != nil {
		t.Fatalf("json list: %v", err)
	}
	verifyKeysExtra(t, &ks)

	ks = keys.InMemoryKeyStore{}
	if err := json.Unmarshal([]byte(jsonMapExtra), &ks); err != nil {
		t.Fatalf("json map: %v", err)
	}
	verifyKeysExtra(t, &ks)
}

func verifyKeysExtra(t *testing.T, ks *keys.InMemoryKeyStore) {
	t.Helper()
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
	tok := keys.NewToken("idval", "user", val)
	if got, want := string(tok.Value()), "secret"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tok.String(), "idval[user]:****"; got != want {
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
	if got, want := tok.ID, ""; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tok.User, ""; got != want {
		t.Errorf("got %v, want %v", got, want)
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

}

func TestMoreContextFunctions(t *testing.T) {
	ctx := context.Background()

	// ContextWithoutKeyStore
	store := keys.NewInMemoryKeyStore()
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

type mockFS struct {
	data map[string][]byte
}

func (m *mockFS) ReadFile(name string) ([]byte, error) {
	if d, ok := m.data[name]; ok {
		return d, nil
	}
	return nil, &json.SyntaxError{} // Just return some error
}

func (m *mockFS) ReadFileCtx(_ context.Context, name string) ([]byte, error) {
	return m.ReadFile(name)
}

func TestKeyOwnerString(t *testing.T) {
	ko := keys.KeyOwner{ID: "id1"}
	if got, want := ko.String(), "id1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	ko.User = "user1"
	if got, want := ko.String(), "id1[user1]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInfoMarshal(t *testing.T) {
	info := keys.NewInfo("id1", "user1", []byte("token1"), map[string]string{"a": "b"})
	buf, err := info.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	// We need to unmarshal into a temporary struct that matches the structure expected by UnmarshalJSON
	// effectively testing round trip if UnmarshalJSON was implemented on Info directly,
	// but Info implementation of UnmarshalJSON (via KeyStore.UnmarshalJSON) handles the structure.
	// Actually Info doesn't have UnmarshalJSON, it's handled by KeyStore or manually.
	// But let's check what MarshalJSON output.

	// Just verify it's valid JSON
	var tmp map[string]any
	if err := json.Unmarshal(buf, &tmp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := tmp["key_id"], "id1"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Test Lazy Loading of Extra
	// Case 1: Extra already set (from NewInfo) - already tested in TestInfo

	// Case 2: Extra from JSON
	jsonStr := `{"key_id": "id1", "user": "user1", "token": "t1", "extra": {"foo": "bar"}}`
	var ks keys.InMemoryKeyStore
	if err := json.Unmarshal([]byte("["+jsonStr+"]"), &ks); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	k1, _ := ks.Get("id1")
	// Extra() should trigger unmarshal
	extra := k1.Extra()
	if extra == nil {
		t.Fatal("expected extra to be not nil")
	}
	// It comes back as map[string]any by default for JSON
	if m, ok := extra.(map[string]any); ok {
		if got, want := m["foo"], "bar"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	} else {
		t.Errorf("expected map[string]any, got %T", extra)
	}

	extraK1, err := json.Marshal(k1)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if got, want := string(extraK1), strings.Replace(jsonStr, " ", "", -1); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Case 3: Extra from YAML
	yamlStr := `
- key_id: id2
  token: t2
  extra:
    bar: baz
`
	var ks2 keys.InMemoryKeyStore
	if err := yaml.Unmarshal([]byte(yamlStr), &ks2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	k2, _ := ks2.Get("id2")
	extra2 := k2.Extra()
	if extra2 == nil {
		t.Fatal("expected extra2 to be not nil")
	}
	// YAML unmarshal might return map[string]any or map[any]any
	// gopkg.in/yaml.v3 usually unmarshals to map[string]any for string keys
	if m, ok := extra2.(map[string]any); ok {
		if got, want := m["bar"], "baz"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	} else {
		// handle potential map[any]any if that happens, though yaml.v3 usually avoids it for string keys
		t.Logf("got %T for extra2", extra2)
	}
}

func TestInMemoryKeyStoreMethods(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("id1", "user1", []byte("t1"), nil))
	ks.Add(keys.NewInfo("id2", "user2", []byte("t2"), nil))

	// KeyOwners
	owners := ks.KeyOwners()
	if got, want := len(owners), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// Order is preserved from append
	if got, want := owners[0].ID, "id1"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Len
	if got, want := ks.Len(), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// MarshalJSON
	buf, err := ks.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	// Verify we can read it back
	var ks2 keys.InMemoryKeyStore
	if err := json.Unmarshal(buf, &ks2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if ks2.Len() != 2 {
		t.Errorf("got %v, want 2", ks2.Len())
	}
}

func TestReadFiles(t *testing.T) {
	ctx := context.Background()
	mfs := &mockFS{
		data: map[string][]byte{
			"keys.json": []byte(jsonList),
			"keys.yaml": []byte(yamlList),
		},
	}

	ks := keys.NewInMemoryKeyStore()
	if err := ks.ReadJSON(ctx, mfs, "keys.json"); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if ks.Len() != 2 {
		t.Errorf("got %v, want 2", ks.Len())
	}

	ks2 := keys.NewInMemoryKeyStore()
	if err := ks2.ReadYAML(ctx, mfs, "keys.yaml"); err != nil {
		t.Fatalf("ReadYAML: %v", err)
	}
	if ks2.Len() != 2 {
		t.Errorf("got %v, want 2", ks2.Len())
	}

	// Error cases
	if err := ks.ReadJSON(ctx, mfs, "missing.json"); err == nil {
		t.Error("expected error for missing file")
	}
	if err := ks.ReadYAML(ctx, mfs, "missing.yaml"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestTokenFromContext(t *testing.T) {
	ctx := context.Background()
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("k1", "u1", []byte("t1"), nil))
	ctx = keys.ContextWithKeyStore(ctx, ks)

	tok, ok := keys.TokenFromContextForID(ctx, "k1")
	if !ok {
		t.Fatal("expected token")
	}
	if got, want := string(tok.Value()), "t1"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, ok = keys.TokenFromContextForID(ctx, "missing")
	if ok {
		t.Error("expected no token")
	}

	ctxNoStore := context.Background()
	_, ok = keys.TokenFromContextForID(ctxNoStore, "k1")
	if ok {
		t.Error("expected no token")
	}
}
