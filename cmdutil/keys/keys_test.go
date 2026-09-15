// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"reflect"
	"sync"
	"testing"
	"time"

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
	k := keys.NewInfo("user1", "key1", []byte("value1"))
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
	ki := keys.NewInfo("user1", "key1", []byte("value1"))
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
	kWithExtra := keys.NewInfo("any_user", "extra_any_key", []byte("secret_val_3"))
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
	tok := keys.NewToken("user", "idval", []byte("abcdefghijk"))

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
	tokShort := keys.NewToken("user", "short", []byte("abc"))
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
	tokMid := keys.NewToken("user", "mid", []byte("abcde"))
	if got, want := tokMid.FirstN(2), "ab******"; got != want {
		t.Errorf("tokMid.FirstN(2) = %q, want %q", got, want)
	}
	if got, want := tokMid.LastN(2), "******de"; got != want {
		t.Errorf("tokMid.LastN(2) = %q, want %q", got, want)
	}

	// Empty token
	tokEmpty := keys.NewToken("user", "empty", []byte(""))
	if got, want := tokEmpty.FirstN(3), ""; got != want {
		t.Errorf("tokEmpty.FirstN(3) = %q, want %q", got, want)
	}
	if got, want := tokEmpty.LastN(3), ""; got != want {
		t.Errorf("tokEmpty.LastN(3) = %q, want %q", got, want)
	}
}

func TestToken(t *testing.T) {
	val := []byte("secret")
	tok := keys.NewToken("user", "idval", val)
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
	info := keys.NewInfo("user", "id", val)
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
	if got, want := parsedWithUser, (keys.KeySpec{User: "user1", ID: "id1"}); got != want {
		t.Errorf("ParseKeySpecValue(id1[user1]) = %+v, want %+v", got, want)
	}

	parsedMalformed := keys.ParseKeySpecValue("id1[user1")
	if got, want := parsedMalformed, (keys.KeySpec{ID: "id1[user1"}); got != want {
		t.Errorf("ParseKeySpecValue(id1[user1) = %+v, want %+v", got, want)
	}
}

func cmpKeyInfo(t *testing.T, a, b keys.Info) {
	t.Helper()
	if a.ID != b.ID || a.User != b.User {
		t.Fatalf("key info mismatch: a=%+v, b=%+v", a, b)
	}
	if !bytes.Equal(a.Token().Value(), b.Token().Value()) {
		t.Fatalf("key info token mismatch: a=%+v, b=%+v", a, b)
	}
}

func cmpKeySpec(t *testing.T, a, b keys.KeySpec) {
	t.Helper()
	if a.ID != b.ID || a.User != b.User {
		t.Fatalf("key spec mismatch: a=%+v, b=%+v", a, b)
	}
}

func TestInMemoryKeyStoreMethods(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	k1 := keys.NewInfo("user1", "id1", []byte("t1"))
	k2 := keys.NewInfo("user1", "id2", []byte("t1.1"))
	k3 := keys.NewInfo("user2", "id2", []byte("t2"))
	ks.Add(k3)
	ks.Add(k2)
	ks.Add(k1)

	// Keys
	kiList := ks.Keys()
	if got, want := len(kiList), 3; got != want {
		t.Errorf("got %v keys, want %v", got, want)
	}
	cmpKeyInfo(t, kiList[0], k1)
	cmpKeyInfo(t, kiList[1], k2)
	cmpKeyInfo(t, kiList[2], k3)

	// KeySpecs
	specs := ks.KeySpecs()
	if got, want := len(specs), 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := ks.Len(), 3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	cmpKeySpec(t, specs[0], k1.KeySpec())
	cmpKeySpec(t, specs[1], k2.KeySpec())
	cmpKeySpec(t, specs[2], k3.KeySpec())

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
	if ks2.Len() != 3 {
		t.Errorf("got %v, want 3", ks2.Len())
	}

	// Verify the actual keys after unmarshalling
	owners2 := ks2.KeySpecs()
	if got, want := len(owners2), 3; got != want {
		t.Fatalf("unmarshaled store has %v keys, want %v", got, want)
	}

	// Assuming order is sorted by ID and User
	cmpKeySpec(t, owners2[0], k1.KeySpec())
	cmpKeySpec(t, owners2[1], k2.KeySpec())
	cmpKeySpec(t, owners2[2], k3.KeySpec())

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
			ks.Add(keys.NewInfo("user0", "id0", []byte("t0")))

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
	ks.Add(keys.NewInfo("user1", "id1", []byte("t1")))
	ks.Add(keys.NewInfo("user2", "id2", []byte("t2")))

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
	ks.Add(keys.NewInfo("user1", "key1", []byte("value1")))
	ks.Add(keys.NewInfo("user2", "key2", []byte("value2")))

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
			ks.Add(keys.NewInfo("user0", "id0", []byte("t0")))

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

// TestGet covers the lookup rules for Get: a specified user is matched
// exactly, and an unspecified user falls back to GetUnique, ie. it only
// succeeds if the id is unique across all users.
func TestGet(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	k1 := keys.NewInfo("user1", "key1", []byte("t1"))
	k2 := keys.NewInfo("user2", "key2", []byte("t2"))
	// key3 is held by two users, so a lookup by id alone is ambiguous.
	k3a := keys.NewInfo("user1", "key3", []byte("t3a"))
	k3b := keys.NewInfo("user2", "key3", []byte("t3b"))
	ks.Add(k1)
	ks.Add(k2)
	ks.Add(k3a)
	ks.Add(k3b)

	// An exact user and id match.
	got, ok := ks.Get("user1", "key1")
	if !ok {
		t.Fatal("expected to find key1/user1")
	}
	cmpKeyInfo(t, got, k1)

	// The right id but the wrong user does not fall back to a search by id.
	if _, ok := ks.Get("user2", "key1"); ok {
		t.Error("key1 belongs to user1, want not found for user2")
	}

	// An empty user falls back to a lookup by id alone, succeeding only when
	// the id is unique.
	got, ok = ks.Get("", "key2")
	if !ok {
		t.Fatal("expected to find the unique key2")
	}
	cmpKeyInfo(t, got, k2)

	// An empty user with an ambiguous id fails, even though the key remains
	// reachable when the user is specified.
	if _, ok := ks.Get("", "key3"); ok {
		t.Error("key3 is held by two users, want not found")
	}
	got, ok = ks.Get("user1", "key3")
	if !ok {
		t.Fatal("expected to find key3/user1")
	}
	cmpKeyInfo(t, got, k3a)

	// An empty user with an unknown id fails.
	if _, ok := ks.Get("", "no-such-key"); ok {
		t.Error("no-such-key: want not found")
	}

	// An empty user and an empty id fails.
	if _, ok := ks.Get("", ""); ok {
		t.Error("empty user and id: want not found")
	}

	// An empty id never matches, not even a key that was itself given an
	// empty id, matching GetUnique's own guarantee, which Get's empty-user
	// fallback shares an implementation with.
	ks.Add(keys.NewInfo("", "", []byte("empty-id")))
	if _, ok := ks.Get("", ""); ok {
		t.Error("empty id: want not found even when such a key exists")
	}
	if _, ok := ks.GetUnique(""); ok {
		t.Error("GetUnique(\"\"): want not found even when such a key exists")
	}
}

// TestGetOwned covers the lookup rules for GetOwned: unlike Get, an empty
// user is never treated as "any user", it is a distinct owner in its own
// right. GetOwned is what callers that must distinguish "this exact key
// exists" from "some key with this id exists" should use, eg. a duplicate
// check before writing a new key.
func TestGetOwned(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	owned := keys.NewInfo("user1", "key1", []byte("owned"))
	unowned := keys.NewInfo("", "key2", []byte("unowned"))
	// key3 is held by both an explicit user and, separately, with no user at
	// all: two distinct keys that happen to share an id.
	ownedShared := keys.NewInfo("user1", "key3", []byte("owned-shared"))
	unownedShared := keys.NewInfo("", "key3", []byte("unowned-shared"))
	ks.Add(owned)
	ks.Add(unowned)
	ks.Add(ownedShared)
	ks.Add(unownedShared)

	// An exact user and id match, owned or not.
	for _, want := range []keys.Info{owned, unowned, ownedShared, unownedShared} {
		got, ok := ks.GetOwned(want.User, want.ID)
		if !ok {
			t.Errorf("%v: not found", want.KeySpec())
			continue
		}
		cmpKeyInfo(t, got, want)
	}

	// Unlike Get, an empty user with an id that is otherwise unique to one
	// owner does NOT match that owner's key: only its own, exact, unowned key
	// counts.
	if _, ok := ks.GetOwned("", "key1"); ok {
		t.Error("key1 has no unowned entry: GetOwned(\"\", \"key1\") got a key, want none")
	}
	// Confirm Get behaves differently here: it does fall back to the unique
	// owner in this case.
	if _, ok := ks.Get("", "key1"); !ok {
		t.Error("Get(\"\", \"key1\"): want the unique owner to be found")
	}

	// Unlike Get, an id held by both an owned and an unowned key is not
	// ambiguous for GetOwned: each is looked up independently by its own
	// exact owner.
	if got, ok := ks.GetOwned("", "key3"); !ok {
		t.Error("GetOwned(\"\", \"key3\"): not found")
	} else {
		cmpKeyInfo(t, got, unownedShared)
	}
	if got, ok := ks.GetOwned("user1", "key3"); !ok {
		t.Error("GetOwned(\"user1\", \"key3\"): not found")
	} else {
		cmpKeyInfo(t, got, ownedShared)
	}
	// Confirm Get is ambiguous here, unlike GetOwned.
	if _, ok := ks.Get("", "key3"); ok {
		t.Error("Get(\"\", \"key3\") is ambiguous: got a key, want none")
	}

	// An id that does not exist at all, and an empty id, are not found either
	// way.
	if _, ok := ks.GetOwned("", "no-such-key"); ok {
		t.Error("no-such-key: want not found")
	}
	if _, ok := ks.GetOwned("user1", ""); ok {
		t.Error("empty id: want not found")
	}
}

func TestGetUnique(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()

	// An empty store has no unique key.
	if _, ok := ks.GetUnique("id1"); ok {
		t.Error("empty store: got a key, want none")
	}

	// A single key with that ID is unique, whether or not it has a user.
	k1 := keys.NewInfo("user1", "id1", []byte("t1"))
	k2 := keys.NewInfo("", "id2", []byte("t2"))
	ks.Add(k1)
	ks.Add(k2)
	for _, want := range []keys.Info{k1, k2} {
		got, ok := ks.GetUnique(want.ID)
		if !ok {
			t.Errorf("%v: not found", want.ID)
			continue
		}
		cmpKeyInfo(t, got, want)
	}

	// An ID that no key has is not found.
	if _, ok := ks.GetUnique("id3"); ok {
		t.Error("id3: got a key, want none")
	}

	// An empty ID is never matched, not even by a key with an empty ID.
	if _, ok := ks.GetUnique(""); ok {
		t.Error("empty id: got a key, want none")
	}
	ks.Add(keys.NewInfo("user1", "", []byte("t0")))
	if _, ok := ks.GetUnique(""); ok {
		t.Error("empty id: got a key, want none")
	}
	ks.Delete("user1", "")

	// A second key with the same ID, belonging to another user, makes the ID
	// ambiguous, even though both keys remain individually retrievable.
	k3 := keys.NewInfo("user2", "id1", []byte("t3"))
	ks.Add(k3)
	if _, ok := ks.GetUnique("id1"); ok {
		t.Error("id1 is held by two users: got a key, want none")
	}
	for _, want := range []keys.Info{k1, k3} {
		got, ok := ks.Get(want.User, want.ID)
		if !ok {
			t.Errorf("%v: not found", want.KeySpec())
			continue
		}
		cmpKeyInfo(t, got, want)
	}
	// The other ID is unaffected.
	if _, ok := ks.GetUnique("id2"); !ok {
		t.Error("id2: not found")
	}

	// Removing the duplicate makes the ID unique again.
	ks.Delete("user2", "id1")
	got, ok := ks.GetUnique("id1")
	if !ok {
		t.Fatal("id1: not found after deleting the duplicate")
	}
	cmpKeyInfo(t, got, k1)

	// Overwriting a key, ie. adding one with the same user and ID, does not
	// make its ID ambiguous.
	ks.Add(keys.NewInfo("user1", "id1", []byte("t1-updated")))
	got, ok = ks.GetUnique("id1")
	if !ok {
		t.Fatal("id1: not found after being overwritten")
	}
	if want := "t1-updated"; string(got.Token().Value()) != want {
		t.Errorf("id1 token: got %v, want %v", string(got.Token().Value()), want)
	}
}

// TestGetUniqueUnmarshalled covers lookups by ID on a store populated by
// unmarshalling, where the keys are not added one at a time.
func TestGetUniqueUnmarshalled(t *testing.T) {
	var ks keys.InMemoryKeyStore
	if err := yaml.Unmarshal([]byte(yamlList), &ks); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, tc := range []struct{ id, token string }{
		{"key1", "value1"},
		{"key2", "value2"},
	} {
		got, ok := ks.GetUnique(tc.id)
		if !ok {
			t.Errorf("%v: not found", tc.id)
			continue
		}
		if want := tc.token; string(got.Token().Value()) != want {
			t.Errorf("%v token: got %v, want %v", tc.id, string(got.Token().Value()), want)
		}
	}
}

// TestGetUniqueConcurrent covers concurrent use of the store, ie. that
// GetUnique holds the lock for the whole of its scan.
func TestGetUniqueConcurrent(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("user1", "id1", []byte("t1")))

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ks.Add(keys.NewInfo("user1", fmt.Sprintf("id%v", i+2), []byte("t")))
		}()
		go func() {
			defer wg.Done()
			// id1 is never added or removed by the writers, so it must always
			// be found.
			if _, ok := ks.GetUnique("id1"); !ok {
				t.Error("id1: not found")
			}
		}()
	}
	wg.Wait()
}

// TestGetOwnedConcurrentNoDeadlock guards against a regression in
// getOwnedLocked, the exact-match lookup shared by Get and GetOwned: it used
// to reacquire ims.mu.RLock() itself even though its callers already held
// it. Per the sync.RWMutex documentation, a second RLock from the same
// goroutine that already holds one can deadlock against a concurrent, blocked
// Lock call, since a pending writer blocks new readers to avoid starvation.
// This runs many concurrent readers (via Get and GetOwned) and writers (via
// Add) and fails if they do not all complete within a generous deadline; it
// reliably hung forever against the buggy implementation.
func TestGetOwnedConcurrentNoDeadlock(t *testing.T) {
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("owner", "id", []byte("t")))

	const iterations = 2000
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				ks.Add(keys.NewInfo(fmt.Sprintf("writer%d", i), fmt.Sprintf("id%d", j), []byte("t")))
			}
		}(i)
	}
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// owner/id is never added or removed by the writers, so both
				// lookups must always find it.
				if _, ok := ks.GetOwned("owner", "id"); !ok {
					t.Error("GetOwned: owner/id not found")
					return
				}
				if _, ok := ks.Get("owner", "id"); !ok {
					t.Error("Get: owner/id not found")
					return
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("readers/writers did not complete: possible deadlock in Get/GetOwned")
	}
}

func testCloneNoTokenNoExtra(t *testing.T) {
	orig := keys.NewInfo("u1", "k1", []byte("secret-token"))
	cloned := orig.CloneNoToken()

	// ID and User preserved
	if got, want := cloned.User, "u1"; got != want {
		t.Errorf("User: got %q, want %q", got, want)
	}
	if got, want := cloned.ID, "k1"; got != want {
		t.Errorf("ID: got %q, want %q", got, want)
	}
	if got, want := cloned.KeySpec(), orig.KeySpec(); got != want {
		t.Errorf("KeySpec: got %+v, want %+v", got, want)
	}
	if got, want := cloned.String(), orig.String(); got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}

	// Token must be stripped in clone
	if len(cloned.Token().Value()) != 0 {
		t.Errorf("cloned token: got %q, want empty", string(cloned.Token().Value()))
	}
	// Original token must be preserved
	if got, want := string(orig.Token().Value()), "secret-token"; got != want {
		t.Errorf("orig token: got %q, want %q", got, want)
	}

	// Extra is nil and unmarshal fails
	if cloned.GetExtra() != nil {
		t.Errorf("GetExtra: got %v, want nil", cloned.GetExtra())
	}
	var extra extraType
	if err := cloned.UnmarshalExtra(&extra); err == nil {
		t.Errorf("UnmarshalExtra: expected error on empty extra, got nil")
	}

	// Clear on empty token shouldn't panic
	cloned.Token().Clear()
}

func testCloneNoTokenWithExtra(t *testing.T) {
	orig := keys.NewInfo("u2", "k2", []byte("secret-val-2"))
	orig.WithExtra(extraType{Scope: "admin"})

	cloned := orig.CloneNoToken()

	// Metadata and token checks
	if got, want := cloned.User, "u2"; got != want {
		t.Errorf("User: got %q, want %q", got, want)
	}
	if got, want := cloned.ID, "k2"; got != want {
		t.Errorf("ID: got %q, want %q", got, want)
	}
	if len(cloned.Token().Value()) != 0 {
		t.Errorf("cloned token: got %q, want empty", string(cloned.Token().Value()))
	}
	if got, want := string(orig.Token().Value()), "secret-val-2"; got != want {
		t.Errorf("orig token: got %q, want %q", got, want)
	}

	// Verify GetExtra and UnmarshalExtra
	if got, want := cloned.GetExtra(), (extraType{Scope: "admin"}); got != want {
		t.Errorf("GetExtra: got %+v, want %+v", got, want)
	}
	verifyExtra(t, cloned, extraType{Scope: "admin"})

	// Serialization roundtrip for JSON
	jsonBuf := marshalJSON(t, &cloned)
	var fromJSON keys.Info
	unmarshalJSON(t, jsonBuf, &fromJSON)
	if len(fromJSON.Token().Value()) != 0 {
		t.Errorf("fromJSON token: got %q, want empty", string(fromJSON.Token().Value()))
	}
	if got, want := fromJSON.User, "u2"; got != want {
		t.Errorf("fromJSON User: got %q, want %q", got, want)
	}
	if got, want := fromJSON.ID, "k2"; got != want {
		t.Errorf("fromJSON ID: got %q, want %q", got, want)
	}
	verifyExtra(t, fromJSON, extraType{Scope: "admin"})

	// Serialization roundtrip for YAML
	yamlBuf := marshalYAML(t, &cloned)
	var fromYAML keys.Info
	unmarshalYAML(t, yamlBuf, &fromYAML)
	if len(fromYAML.Token().Value()) != 0 {
		t.Errorf("fromYAML token: got %q, want empty", string(fromYAML.Token().Value()))
	}
	if got, want := fromYAML.User, "u2"; got != want {
		t.Errorf("fromYAML User: got %q, want %q", got, want)
	}
	if got, want := fromYAML.ID, "k2"; got != want {
		t.Errorf("fromYAML ID: got %q, want %q", got, want)
	}
	verifyExtra(t, fromYAML, extraType{Scope: "admin"})
}

func testCloneNoTokenPrivateFields(t *testing.T) {
	type extraWithPrivate struct {
		Scope   string `json:"scope" yaml:"scope"`
		private int
	}
	orig := keys.NewInfo("u_priv", "k_priv", []byte("token_priv"))
	orig.WithExtra(extraWithPrivate{Scope: "audit", private: 42})

	cloned := orig.CloneNoToken()
	if len(cloned.Token().Value()) != 0 {
		t.Errorf("cloned token: got %q, want empty", string(cloned.Token().Value()))
	}
	var got extraWithPrivate
	if err := cloned.UnmarshalExtra(&got); err != nil {
		t.Fatalf("UnmarshalExtra: %v", err)
	}
	if got.Scope != "audit" || got.private != 42 {
		t.Errorf("got %+v, want Scope=audit private=42", got)
	}
}

func testCloneNoTokenYAML(t *testing.T) {
	yamlData := `user: yaml_user
key_id: yaml_key
token: "yaml_secret_val"
extra:
  scope: operator
`
	var orig keys.Info
	unmarshalYAML(t, []byte(yamlData), &orig)
	if got, want := string(orig.Token().Value()), "yaml_secret_val"; got != want {
		t.Fatalf("orig token: got %q, want %q", got, want)
	}

	cloned := orig.CloneNoToken()

	if got, want := cloned.User, "yaml_user"; got != want {
		t.Errorf("User: got %q, want %q", got, want)
	}
	if got, want := cloned.ID, "yaml_key"; got != want {
		t.Errorf("ID: got %q, want %q", got, want)
	}
	if len(cloned.Token().Value()) != 0 {
		t.Errorf("cloned token: got %q, want empty", string(cloned.Token().Value()))
	}
	if got, want := string(orig.Token().Value()), "yaml_secret_val"; got != want {
		t.Errorf("orig token: got %q, want %q", got, want)
	}

	verifyExtra(t, cloned, extraType{Scope: "operator"})

	// Serialization roundtrip for YAML
	yamlBuf := marshalYAML(t, &cloned)
	var fromYAML keys.Info
	unmarshalYAML(t, yamlBuf, &fromYAML)
	if len(fromYAML.Token().Value()) != 0 {
		t.Errorf("fromYAML token: got %q, want empty", string(fromYAML.Token().Value()))
	}
	verifyExtra(t, fromYAML, extraType{Scope: "operator"})

	// Serialization roundtrip to JSON
	jsonBuf := marshalJSON(t, &cloned)
	var fromJSON keys.Info
	unmarshalJSON(t, jsonBuf, &fromJSON)
	if len(fromJSON.Token().Value()) != 0 {
		t.Errorf("fromJSON token: got %q, want empty", string(fromJSON.Token().Value()))
	}
	verifyExtra(t, fromJSON, extraType{Scope: "operator"})
}

func testCloneNoTokenJSON(t *testing.T) {
	jsonData := `{"user": "json_user", "key_id": "json_key", "token": "json_secret_val", "extra": {"scope": "deployer"}}`
	var orig keys.Info
	unmarshalJSON(t, []byte(jsonData), &orig)
	if got, want := string(orig.Token().Value()), "json_secret_val"; got != want {
		t.Fatalf("orig token: got %q, want %q", got, want)
	}

	cloned := orig.CloneNoToken()

	if got, want := cloned.User, "json_user"; got != want {
		t.Errorf("User: got %q, want %q", got, want)
	}
	if got, want := cloned.ID, "json_key"; got != want {
		t.Errorf("ID: got %q, want %q", got, want)
	}
	if len(cloned.Token().Value()) != 0 {
		t.Errorf("cloned token: got %q, want empty", string(cloned.Token().Value()))
	}
	if got, want := string(orig.Token().Value()), "json_secret_val"; got != want {
		t.Errorf("orig token: got %q, want %q", got, want)
	}

	verifyExtra(t, cloned, extraType{Scope: "deployer"})

	// Serialization roundtrip for JSON
	jsonBuf := marshalJSON(t, &cloned)
	var fromJSON keys.Info
	unmarshalJSON(t, jsonBuf, &fromJSON)
	if len(fromJSON.Token().Value()) != 0 {
		t.Errorf("fromJSON token: got %q, want empty", string(fromJSON.Token().Value()))
	}
	verifyExtra(t, fromJSON, extraType{Scope: "deployer"})

	// Serialization roundtrip to YAML
	yamlBuf := marshalYAML(t, &cloned)
	var fromYAML keys.Info
	unmarshalYAML(t, yamlBuf, &fromYAML)
	if len(fromYAML.Token().Value()) != 0 {
		t.Errorf("fromYAML token: got %q, want empty", string(fromYAML.Token().Value()))
	}
	verifyExtra(t, fromYAML, extraType{Scope: "deployer"})
}

func testCloneNoTokenStore(t *testing.T) {
	orig := keys.NewInfo("u_store", "k_store", []byte("store-secret"))
	orig.WithExtra(extraType{Scope: "stored"})

	cloned := orig.CloneNoToken()

	ks := keys.NewInMemoryKeyStore()
	ks.Add(cloned)

	got, ok := ks.Get("u_store", "k_store")
	if !ok {
		t.Fatal("key not found in store")
	}
	if len(got.Token().Value()) != 0 {
		t.Errorf("stored token: got %q, want empty", string(got.Token().Value()))
	}
	verifyExtra(t, got, extraType{Scope: "stored"})
}

func testCloneNoTokenMutation(t *testing.T) {
	tokBytes := []byte("original-token-bytes")
	orig := keys.NewInfo("u_indep", "k_indep", tokBytes)
	orig.WithExtra(extraType{Scope: "original_scope"})

	cloned := orig.CloneNoToken()

	// Modifying original extra
	orig.WithExtra(extraType{Scope: "modified_scope"})

	// Cloned extra should remain unchanged
	verifyExtra(t, cloned, extraType{Scope: "original_scope"})

	// Modifying original fields
	orig.User = "u_modified"
	orig.ID = "k_modified"
	if got, want := cloned.User, "u_indep"; got != want {
		t.Errorf("cloned User after orig mutation: got %q, want %q", got, want)
	}
	if got, want := cloned.ID, "k_indep"; got != want {
		t.Errorf("cloned ID after orig mutation: got %q, want %q", got, want)
	}

	// Cloned remains with empty token
	if len(cloned.Token().Value()) != 0 {
		t.Errorf("cloned token: got %q, want empty", string(cloned.Token().Value()))
	}
}

func TestCloneNoToken(t *testing.T) {
	t.Run("NoExtra", testCloneNoTokenNoExtra)
	t.Run("WithExtra_Direct", testCloneNoTokenWithExtra)
	t.Run("WithExtra_PrivateFields", testCloneNoTokenPrivateFields)
	t.Run("ExtraFromYAML", testCloneNoTokenYAML)
	t.Run("ExtraFromJSON", testCloneNoTokenJSON)
	t.Run("StoreIntegration", testCloneNoTokenStore)
	t.Run("MutationIndependence", testCloneNoTokenMutation)
}
