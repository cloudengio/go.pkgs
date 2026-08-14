// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
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
	k1, ok := ks.Get("user1", "key1")
	if !ok {
		t.Fatalf("key1 not found")
	}
	verifyKey(t, k1, 1)
	k2, ok := ks.Get("user2", "key2")
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
	k1, ok := ks.Get("user1", "key1")
	if !ok {
		t.Fatalf("key1 not found")
	}

	verifyExtra(t, k1, extraType{Scope: "read"})

	k2, ok := ks.Get("user2", "key2")
	if !ok {
		t.Fatalf("key2 not found")
	}
	verifyExtra(t, k2, extraType{Scope: "write"})
}

func verifyAppendedKeys(t *testing.T, ks *keys.InMemoryKeyStore, checkExtra bool) {
	t.Helper()
	expectedLen := 3 // 1 initial + 2 unique keys (which get updated)
	if got, want := ks.Len(), expectedLen; got != want {
		t.Fatalf("got len %v, want %v", got, want)
	}
	if _, ok := ks.Get("user0", "id0"); !ok {
		t.Error("key id0 not found after append")
	}
	if _, ok := ks.Get("user1", "key1"); !ok {
		t.Error("key key1 not found after append")
	}
	if _, ok := ks.Get("user2", "key2"); !ok {
		t.Error("key key2 not found after append")
	}
	if checkExtra {
		verifyKeysExtra(t, ks)
	}
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

	// Test Extra marshaling/unmarshaling roundtrip across representations
	// 1. extraYAML -> MarshalJSON -> UnmarshalJSON -> verify extra and token
	kiYAML := `key_id: yaml_extra_key
token: "secret_val_1"
user: yaml_user
extra:
  scope: admin
`
	var kYAML keys.Info
	unmarshalYAML(t, []byte(kiYAML), &kYAML)
	if got, want := string(kYAML.Token().Value()), "secret_val_1"; got != want {
		t.Errorf("token: got %v, want %v", got, want)
	}
	jsonBuf := marshalJSON(t, &kYAML)
	var kFromYAMLtoHex keys.Info
	unmarshalJSON(t, jsonBuf, &kFromYAMLtoHex)
	if got, want := string(kFromYAMLtoHex.Token().Value()), "secret_val_1"; got != want {
		t.Errorf("roundtrip token: got %v, want %v", got, want)
	}
	verifyExtra(t, kFromYAMLtoHex, extraType{Scope: "admin"})

	// 2. extraJSON -> MarshalYAML -> UnmarshalYAML -> verify extra and token
	kiJSON := `{"key_id": "json_extra_key", "token": "secret_val_2", "user": "json_user", "extra": {"scope": "operator"}}`
	var kJSON keys.Info
	unmarshalJSON(t, []byte(kiJSON), &kJSON)
	if got, want := string(kJSON.Token().Value()), "secret_val_2"; got != want {
		t.Errorf("token: got %v, want %v", got, want)
	}
	yamlBuf := marshalYAML(t, &kJSON)
	var kFromJSONtoYAML keys.Info
	unmarshalYAML(t, yamlBuf, &kFromJSONtoYAML)
	if got, want := string(kFromJSONtoYAML.Token().Value()), "secret_val_2"; got != want {
		t.Errorf("roundtrip token: got %v, want %v", got, want)
	}
	verifyExtra(t, kFromJSONtoYAML, extraType{Scope: "operator"})

	// 3. WithExtra (extraAny) -> MarshalYAML -> UnmarshalYAML
	kWithExtra := keys.NewInfo("extra_any_key", "any_user", []byte("secret_val_3"))
	kWithExtra.WithExtra(extraType{Scope: "custom"})
	yamlBuf2 := marshalYAML(t, &kWithExtra)
	var kFromAnyToYAML keys.Info
	unmarshalYAML(t, yamlBuf2, &kFromAnyToYAML)
	if got, want := string(kFromAnyToYAML.Token().Value()), "secret_val_3"; got != want {
		t.Errorf("token: got %v, want %v", got, want)
	}
	verifyExtra(t, kFromAnyToYAML, extraType{Scope: "custom"})

	// 4. WithExtra (extraAny) -> MarshalJSON -> UnmarshalJSON
	jsonBuf2 := marshalJSON(t, &kWithExtra)
	var kFromAnyToJSON keys.Info
	unmarshalJSON(t, jsonBuf2, &kFromAnyToJSON)
	if got, want := string(kFromAnyToJSON.Token().Value()), "secret_val_3"; got != want {
		t.Errorf("token: got %v, want %v", got, want)
	}
	verifyExtra(t, kFromAnyToJSON, extraType{Scope: "custom"})
}

func TestTokenRedaction(t *testing.T) {
	tok := keys.NewToken("idval", "user", []byte("abcdefghijk"))

	// Test FirstN
	testsFirstN := []struct {
		keep int
		want string
	}{
		{keep: 0, want: "***********"},
		{keep: -1, want: "***********"},
		{keep: 1, want: "a******"},
		{keep: 3, want: "abc******"},
		{keep: 6, want: "abcdef******"},
		{keep: 7, want: "abcdef******"}, // capped at DefaultRedactionLimit (6)
		{keep: 100, want: "abcdef******"},
	}
	for _, tc := range testsFirstN {
		if got := tok.FirstN(tc.keep); got != tc.want {
			t.Errorf("tok.FirstN(%d) = %q, want %q", tc.keep, got, tc.want)
		}
	}

	// Test LastN
	testsLastN := []struct {
		keep int
		want string
	}{
		{keep: 0, want: "***********"},
		{keep: -1, want: "***********"},
		{keep: 1, want: "******k"},
		{keep: 3, want: "******ijk"},
		{keep: 6, want: "******fghijk"},
		{keep: 7, want: "******fghijk"}, // capped at DefaultRedactionLimit (6)
		{keep: 100, want: "******fghijk"},
	}
	for _, tc := range testsLastN {
		if got := tok.LastN(tc.keep); got != tc.want {
			t.Errorf("tok.LastN(%d) = %q, want %q", tc.keep, got, tc.want)
		}
	}

	// Short token tests (token length <= keep)
	tokShort := keys.NewToken("short", "user", []byte("abc"))
	if got, want := tokShort.FirstN(3), "***"; got != want {
		t.Errorf("tokShort.FirstN(3) = %q, want %q", got, want)
	}
	if got, want := tokShort.FirstN(4), "***"; got != want {
		t.Errorf("tokShort.FirstN(4) = %q, want %q", got, want)
	}
	if got, want := tokShort.LastN(3), "***"; got != want {
		t.Errorf("tokShort.LastN(3) = %q, want %q", got, want)
	}
	if got, want := tokShort.LastN(4), "***"; got != want {
		t.Errorf("tokShort.LastN(4) = %q, want %q", got, want)
	}

	// Token shorter than 6 characters with valid keep < len
	tokMid := keys.NewToken("mid", "user", []byte("abcde"))
	if got, want := tokMid.FirstN(2), "ab******"; got != want {
		t.Errorf("tokMid.FirstN(2) = %q, want %q", got, want)
	}
	if got, want := tokMid.LastN(2), "******de"; got != want {
		t.Errorf("tokMid.LastN(2) = %q, want %q", got, want)
	}

	// Empty token
	tokEmpty := keys.NewToken("empty", "user", []byte(""))
	if got, want := tokEmpty.FirstN(3), ""; got != want {
		t.Errorf("tokEmpty.FirstN(3) = %q, want %q", got, want)
	}
	if got, want := tokEmpty.LastN(3), ""; got != want {
		t.Errorf("tokEmpty.LastN(3) = %q, want %q", got, want)
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

	// Make sure underlying token value is cleared
	for _, b := range tok.Value() {
		if b != 0 {
			t.Errorf("token was not cleared")
		}
	}
	for _, b := range val {
		if b != 0 {
			t.Errorf("token was not cleared")
		}
	}
	// UserID and ID need not be cleared
	if got, want := tok.ID, tok.ID; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := tok.User, tok.User; got != want {
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
	if got, want := info.GetExtra().(map[string]string)["a"], "b"; got != want {
		t.Errorf("GetExtra: got %v, want %v", got, want)
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

type mockWriteFS struct {
	data map[string][]byte
}

func (m *mockWriteFS) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.data[name] = append([]byte(nil), data...)
	return nil
}

func (m *mockWriteFS) WriteFileCtx(_ context.Context, name string, data []byte, perm fs.FileMode) error {
	return m.WriteFile(name, data, perm)
}

func TestKeySpecString(t *testing.T) {
	ks := keys.KeySpec{ID: "id1"}
	if got, want := ks.String(), "id1"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	ks.User = "user1"
	if got, want := ks.String(), "id1[user1]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// ParseKeySpecValue tests
	parsed := keys.ParseKeySpecValue("id1")
	if got, want := parsed, (keys.KeySpec{ID: "id1"}); got != want {
		t.Errorf("ParseKeySpecValue(id1) = %+v, want %+v", got, want)
	}

	parsedWithUser := keys.ParseKeySpecValue("id1[user1]")
	if got, want := parsedWithUser, (keys.KeySpec{ID: "id1", User: "user1"}); got != want {
		t.Errorf("ParseKeySpecValue(id1[user1]) = %+v, want %+v", got, want)
	}

	parsedMalformed := keys.ParseKeySpecValue("id1[user1")
	if got, want := parsedMalformed, (keys.KeySpec{ID: "id1[user1"}); got != want {
		t.Errorf("ParseKeySpecValue(id1[user1) = %+v, want %+v", got, want)
	}
}

func TestInMemoryKeyStoreMethods(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("id1", "user1", []byte("t1")))
	ks.Add(keys.NewInfo("id2", "user2", []byte("t2")))

	// Keys
	kiList := ks.Keys()
	if got, want := len(kiList), 2; got != want {
		t.Errorf("got %v keys, want %v", got, want)
	}
	if got, want := kiList[0].ID, "id1"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := kiList[1].ID, "id2"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	// KeySpecs
	specs := ks.KeySpecs()
	if got, want := len(specs), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	// Order is sorted by ID and User
	if got, want := specs[0].ID, "id1"; got != want {
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
	owners2 := ks2.KeySpecs()
	if got, want := len(owners2), 2; got != want {
		t.Fatalf("unmarshaled store has %v keys, want %v", got, want)
	}

	// Assuming order is sorted by ID and User
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
	k1Unmarshaled, _ := ks2.Get("user1", "id1")
	if got, want := string(k1Unmarshaled.Token().Value()), "t1"; got != want {
		t.Errorf("unmarshaled key 1 token: got %v, want %v", got, want)
	}
	k2Unmarshaled, _ := ks2.Get("user2", "id2")
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

func TestAppendUnmarshal(t *testing.T) {
	testCases := []struct {
		name   string
		data   []string
		isYAML bool
	}{
		{"JSON list", []string{jsonList, jsonListExtra}, false},
		{"JSON map", []string{jsonMap, jsonMapExtra}, false},
		{"JSON mixed", []string{jsonList, jsonMapExtra}, false},
		{"YAML list", []string{yamlList, yamlListExtra}, true},
		{"YAML map", []string{yamlMap, yamlMapExtra}, true},
		{"YAML mixed", []string{yamlList, yamlMapExtra}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ks := keys.NewInMemoryKeyStore()
			ks.Add(keys.NewInfo("id0", "user0", []byte("t0")))

			for _, data := range tc.data {
				var err error
				if tc.isYAML {
					err = yaml.Unmarshal([]byte(data), ks)
				} else {
					err = ks.UnmarshalJSON([]byte(data))
				}
				if err != nil {
					t.Fatalf("Unmarshal: %v", err)
				}
			}

			verifyAppendedKeys(t, ks, len(tc.data) > 1)
		})
	}
}

func TestDelete(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("id1", "user1", []byte("t1")))
	ks.Add(keys.NewInfo("id2", "user2", []byte("t2")))

	ks.Delete("user1", "id1")
	if got, want := ks.Len(), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if _, ok := ks.Get("user1", "id1"); ok {
		t.Error("deleted key still present")
	}
	if _, ok := ks.Get("user2", "id2"); !ok {
		t.Error("remaining key not found")
	}

	// Delete of a non-existent key is a no-op.
	ks.Delete("user1", "id1")
	if got, want := ks.Len(), 1; got != want {
		t.Errorf("after no-op delete: got %v, want %v", got, want)
	}
}

func TestWriteFiles(t *testing.T) {
	ctx := context.Background()

	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("key1", "user1", []byte("value1")))
	ks.Add(keys.NewInfo("key2", "user2", []byte("value2")))

	wfs := &mockWriteFS{data: make(map[string][]byte)}
	rfs := &mockFS{data: wfs.data}

	// WriteJSON round-trip.
	if err := ks.WriteJSON(ctx, wfs, "keys.json", 0o600); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	ks2 := keys.NewInMemoryKeyStore()
	if err := ks2.ReadJSON(ctx, rfs, "keys.json"); err != nil {
		t.Fatalf("ReadJSON after WriteJSON: %v", err)
	}
	verifyKeys(t, ks2)

	// WriteYAML round-trip.
	if err := ks.WriteYAML(ctx, wfs, "keys.yaml", 0o600); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}
	ks3 := keys.NewInMemoryKeyStore()
	if err := ks3.ReadYAML(ctx, rfs, "keys.yaml"); err != nil {
		t.Fatalf("ReadYAML after WriteYAML: %v", err)
	}
	verifyKeys(t, ks3)

	// Empty name is an error for both formats.
	if err := ks.WriteJSON(ctx, wfs, "", 0o600); err == nil {
		t.Error("WriteJSON: expected error for empty name")
	}
	if err := ks.WriteYAML(ctx, wfs, "", 0o600); err == nil {
		t.Error("WriteYAML: expected error for empty name")
	}
}

func TestAppendRead(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name     string
		data     []string
		filename []string
		isYAML   bool
	}{
		{"JSON list", []string{jsonList, jsonListExtra}, []string{"keys1.json", "keys2.json"}, false},
		{"JSON map", []string{jsonMap, jsonMapExtra}, []string{"keys1.json", "keys2.json"}, false},
		{"JSON mixed", []string{jsonList, jsonMapExtra}, []string{"keys1.json", "keys2.json"}, false},
		{"YAML list", []string{yamlList, yamlListExtra}, []string{"keys1.yaml", "keys2.yaml"}, true},
		{"YAML map", []string{yamlMap, yamlMapExtra}, []string{"keys1.yaml", "keys2.yaml"}, true},
		{"YAML mixed", []string{yamlList, yamlMapExtra}, []string{"keys1.yaml", "keys2.yaml"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ks := keys.NewInMemoryKeyStore()
			ks.Add(keys.NewInfo("id0", "user0", []byte("t0")))

			mfs := &mockFS{
				data: make(map[string][]byte),
			}
			for i, data := range tc.data {
				mfs.data[tc.filename[i]] = []byte(data)
			}

			for _, fname := range tc.filename {
				var err error
				if tc.isYAML {
					err = ks.ReadYAML(ctx, mfs, fname)
				} else {
					err = ks.ReadJSON(ctx, mfs, fname)
				}
				if err != nil {
					t.Fatalf("Read failed: %v", err)
				}
			}

			verifyAppendedKeys(t, ks, len(tc.data) > 1)
		})
	}
}
