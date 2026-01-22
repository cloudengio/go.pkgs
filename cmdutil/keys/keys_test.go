// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"cloudeng.io/cmdutil/keys"
	"gopkg.in/yaml.v3"
)

const (
	yamlList = `
- key_id: key1
  token: "value1"
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

func unmarshalJSON(t *testing.T, buf []byte, tmp any) {
	t.Helper()
	if err := json.Unmarshal(buf, tmp); err != nil {
		t.Fatalf("UnmarshalJSON: %s: %v", string(buf), err)
	}
}

func marshalJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	return b
}

func unmarshalYAML(t *testing.T, buf []byte, tmp any) {
	t.Helper()
	if err := yaml.Unmarshal(buf, tmp); err != nil {
		t.Fatalf("UnmarshalYAML: %s: %v", string(buf), err)
	}
}

func marshalYAML(t *testing.T, v any) []byte {
	t.Helper()
	b, err := yaml.Marshal(v)
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	return b
}

func TestYAMLKeyInfo(t *testing.T) {
	ki := `key_id: key1
token: “value1”
user: user1
`
	// Unmarshal
	var k keys.Info
	unmarshalYAML(t, []byte(ki), &k)
	verifyKey(t, k, 1)

	// Round trip
	out := marshalYAML(t, &k)
	var k1 keys.Info
	unmarshalYAML(t, out, &k1)
	verifyKey(t, k1, 1)

	// Unmarshal with extra
	kiExtra := `key_id: key1
token: value1
user: user1
extra:
  scope: read
`

	unmarshalYAML(t, []byte(kiExtra), &k)
	verifyKey(t, k, 1)
	verifyExtra(t, k, extraType{Scope: "read"})

	// Round trip with extra
	out = marshalYAML(t, &k)
	var k2 keys.Info
	unmarshalYAML(t, out, &k2)
	verifyKey(t, k2, 1)
	verifyExtra(t, k2, extraType{Scope: "read"})

	// YAML <-> JSON
	kiExtra = `key_id: key2
token: value2
user: user2
extra:
  scope: "write"
`
	unmarshalYAML(t, []byte(kiExtra), &k)
	verifyKey(t, k, 2)
	verifyExtra(t, k, extraType{Scope: "write"})

	out = marshalJSON(t, &k)
	var k3 keys.Info
	unmarshalJSON(t, out, &k3)
	verifyKey(t, k3, 2)
	verifyExtra(t, k3, extraType{Scope: "write"})
}

func TestJSONKeyInfo(t *testing.T) {
	ki := `{"key_id": "key1", "token": "value1", "user": "user1"}`
	var k keys.Info

	// Unmarshal
	unmarshalJSON(t, []byte(ki), &k)
	verifyKey(t, k, 1)

	// Round trip
	buf := marshalJSON(t, &k)
	var k1 keys.Info
	unmarshalJSON(t, buf, &k1)
	verifyKey(t, k1, 1)

	// Unmarshal with extra
	kiExtra := `{"key_id": "key1", "token": "value1", "user": "user1", "extra": {"scope": "read"}}`
	unmarshalJSON(t, []byte(kiExtra), &k)
	verifyKey(t, k, 1)
	verifyExtra(t, k, extraType{Scope: "read"})

	// Round trip with extra
	buf = marshalJSON(t, &k)
	var k2 keys.Info
	unmarshalJSON(t, buf, &k2)
	verifyKey(t, k2, 1)
	verifyExtra(t, k2, extraType{Scope: "read"})

	// JSON <-> YAML
	kiExtra = `{"key_id": "key2", "token": "value2", "user": "user2", "extra": {"scope": "write"}}`
	unmarshalJSON(t, []byte(kiExtra), &k)
	verifyKey(t, k, 2)
	verifyExtra(t, k, extraType{Scope: "write"})

	buf = marshalYAML(t, &k)
	var k3 keys.Info
	unmarshalYAML(t, buf, &k3)
	verifyKey(t, k3, 2)
	verifyExtra(t, k3, extraType{Scope: "write"})
}

func TestNewKey(t *testing.T) {
	k := keys.NewInfo("key1", "user1", []byte("value1"))
	verifyKey(t, k, 1)
	out := marshalJSON(t, k)

	var k1 keys.Info
	unmarshalJSON(t, out, &k1)
	verifyKey(t, k1, 1)

	out = marshalYAML(t, k)
	var k2 keys.Info
	unmarshalYAML(t, out, &k2)
	verifyKey(t, k2, 1)

	k.WithExtra(extraType{Scope: "read"})
	out = marshalJSON(t, k)

	var k3 keys.Info
	unmarshalJSON(t, out, &k3)
	verifyKey(t, k3, 1)
	verifyExtra(t, k3, extraType{Scope: "read"})

	var k4 keys.Info
	unmarshalYAML(t, out, &k4)
	verifyKey(t, k4, 1)
	verifyExtra(t, k4, extraType{Scope: "read"})
}

func TestYAMLStore(t *testing.T) {
	var ks keys.InMemoryKeyStore
	unmarshalYAML(t, []byte(yamlList), &ks) // list
	verifyKeys(t, &ks)

	// round trip
	buf := marshalYAML(t, &ks)
	ks = keys.InMemoryKeyStore{}
	unmarshalYAML(t, buf, &ks)
	verifyKeys(t, &ks)

	ks = keys.InMemoryKeyStore{}
	unmarshalYAML(t, []byte(yamlMap), &ks) // map
	verifyKeys(t, &ks)
}

func TestJSONStore(t *testing.T) {
	var ks keys.InMemoryKeyStore
	unmarshalJSON(t, []byte(jsonList), &ks) // list
	verifyKeys(t, &ks)

	// round trip
	buf := marshalJSON(t, &ks)
	ks = keys.InMemoryKeyStore{}
	unmarshalJSON(t, buf, &ks)
	verifyKeys(t, &ks)

	ks = keys.InMemoryKeyStore{}
	unmarshalJSON(t, []byte(jsonMap), &ks) // map
	verifyKeys(t, &ks)
}

func verifyKey(t *testing.T, k keys.Info, i int) {
	t.Helper()
	if got, want := k.ID, fmt.Sprintf("key%d", i); got != want {
		t.Errorf("key%d ID: got %v, want %v", i, got, want)
	}
	if got, want := string(k.Token().Value()), fmt.Sprintf("value%d", i); got != want {
		t.Errorf("key%d: got %v, want %v", i, got, want)
	}

	if got, want := k.User, fmt.Sprintf("user%d", i); got != want {
		t.Errorf("key%d user: got %v, want %v", i, got, want)
	}
}

func verifyKeys(t *testing.T, ks *keys.InMemoryKeyStore) {
	t.Helper()
	k1, ok := ks.Get("key1")
	if !ok {
		t.Fatalf("key1 not found")
	}
	verifyKey(t, k1, 1)
	k2, ok := ks.Get("key2")
	if !ok {
		t.Fatalf("key2 not found")
	}
	verifyKey(t, k2, 2)
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
	unmarshalYAML(t, []byte(yamlListExtra), &ks)
	verifyKeysExtra(t, &ks)

	ks = keys.InMemoryKeyStore{}
	unmarshalYAML(t, []byte(yamlMapExtra), &ks)
	verifyKeysExtra(t, &ks)
}

func TestUnmarshalJSONExtra(t *testing.T) {
	var ks keys.InMemoryKeyStore
	unmarshalJSON(t, []byte(jsonListExtra), &ks)
	verifyKeysExtra(t, &ks)

	ks = keys.InMemoryKeyStore{}
	unmarshalJSON(t, []byte(jsonMapExtra), &ks)
	verifyKeysExtra(t, &ks)
}

func verifyExtra[T any](t *testing.T, k keys.Info, e T) {
	t.Helper()
	var want T
	if err := k.UnmarshalExtra(&want); err != nil {
		t.Fatalf("key1 extra: %v", err)
	}
	if !reflect.DeepEqual(want, e) {
		t.Errorf("key1 extra: got %+v, want %+v", want, e)
	}
}

func verifyKeysExtra(t *testing.T, ks *keys.InMemoryKeyStore) {
	t.Helper()
	k1, ok := ks.Get("key1")
	if !ok {
		t.Fatalf("key1 not found")
	}

	verifyExtra(t, k1, extraType{Scope: "read"})

	k2, ok := ks.Get("key2")
	if !ok {
		t.Fatalf("key2 not found")
	}
	verifyExtra(t, k2, extraType{Scope: "write"})
}

func TestExtraWithPrivateFields(t *testing.T) {
	ki := keys.NewInfo("key1", "user1", []byte("value1"))
	type extraTypeWithPrivate struct {
		Scope   string `json:"scope" yaml:"scope"`
		private int
	}

	// Verify that private fields can be retrieved for extra
	// values set directly.
	ki.WithExtra(extraTypeWithPrivate{Scope: "read", private: 1})
	var e extraTypeWithPrivate
	if err := ki.UnmarshalExtra(&e); err != nil {
		t.Fatalf("key1 extra: %v", err)
	}
	if got, want := e.Scope, "read"; got != want {
		t.Errorf("key1 extra: got %v, want %v", got, want)
	}
	if got, want := e.private, 1; got != want {
		t.Errorf("key1 extra: got %v, want %v", got, want)
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
	info := keys.NewInfo("id", "user", val)
	info.WithExtra(extra)

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
	var e map[string]string
	if err := info.UnmarshalExtra(&e); err != nil {
		t.Fatalf("info extra: %v", err)
	}
	if got, want := e["a"], "b"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// Verify input was cleared
	if string(val) == "secret" {
		t.Errorf("input slice was not cleared")
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

func TestInMemoryKeyStoreMethods(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("id1", "user1", []byte("t1")))
	ks.Add(keys.NewInfo("id2", "user2", []byte("t2")))

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

	// Verify the actual keys after unmarshalling
	owners2 := ks2.KeyOwners()
	if got, want := len(owners2), 2; got != want {
		t.Fatalf("unmarshaled store has %v keys, want %v", got, want)
	}

	// Assuming order is preserved (which it should be for a slice)
	if got, want := owners2[0].ID, "id1"; got != want {
		t.Errorf("unmarshaled key 1 ID: got %v, want %v", got, want)
	}
	if got, want := owners2[0].User, "user1"; got != want {
		t.Errorf("unmarshaled key 1 User: got %v, want %v", got, want)
	}
	if got, want := owners2[1].ID, "id2"; got != want {
		t.Errorf("unmarshaled key 2 ID: got %v, want %v", got, want)
	}
	if got, want := owners2[1].User, "user2"; got != want {
		t.Errorf("unmarshaled key 2 User: got %v, want %v", got, want)
	}

	// Also verify that the token values are preserved (lazy loaded)
	k1Unmarshaled, _ := ks2.Get("id1")
	if got, want := string(k1Unmarshaled.Token().Value()), "t1"; got != want {
		t.Errorf("unmarshaled key 1 token: got %v, want %v", got, want)
	}
	k2Unmarshaled, _ := ks2.Get("id2")
	if got, want := string(k2Unmarshaled.Token().Value()), "t2"; got != want {
		t.Errorf("unmarshaled key 2 token: got %v, want %v", got, want)
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
