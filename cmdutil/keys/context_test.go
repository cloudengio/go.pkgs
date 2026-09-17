// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"cloudeng.io/cmdutil/keys"
)

func TestContextFunctions(t *testing.T) {
	store := keys.NewInMemoryKeyStore()
	k1 := keys.NewInfo("ctx-user", "ctx-key", []byte("ctx-token"))
	store.Add(k1)

	ctx := keys.ContextWithKeyStore(context.Background(), store)

	// Test successful retrieval
	retrievedKey, ok := keys.KeyInfoFromContext(ctx, "ctx-user", "ctx-key")
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
	_, ok = keys.KeyInfoFromContext(ctx, "ctx-user", "non-existent")
	if ok {
		t.Error("did not expect to find non-existent key in context")
	}

	// Test retrieval from a context without the store
	emptyCtx := context.Background()
	_, ok = keys.KeyInfoFromContext(emptyCtx, "ctx-user", "ctx-key")
	if ok {
		t.Error("did not expect to find key in empty context")
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
	k1 := keys.NewInfo("u1", "k1", []byte("t1"))
	ctxWithKey := keys.ContextWithKey(ctx, k1)

	// Should have created a store and added the key
	storeFromCtx, ok := keys.KeyStoreFromContext(ctxWithKey)
	if !ok {
		t.Fatal("expected store to be created")
	}

	gotKey, ok := storeFromCtx.Get("u1", "k1")
	if !ok {
		t.Fatal("expected key to be in store")
	}
	if gotKey.ID != "k1" {
		t.Errorf("got %v, want k1", gotKey.ID)
	}

	// Add another key to existing store
	k2 := keys.NewInfo("u2", "k2", []byte("t2"))
	ctxWithKey2 := keys.ContextWithKey(ctxWithKey, k2)

	storeFromCtx2, ok := keys.KeyStoreFromContext(ctxWithKey2)
	if !ok {
		t.Fatal("expected store")
	}
	if _, ok := storeFromCtx2.Get("u2", "k2"); !ok {
		t.Fatal("expected k2")
	}
	if _, ok := storeFromCtx2.Get("u1", "k1"); !ok {
		t.Fatal("expected k1 to still be there")
	}
}

func TestTokenFromContext(t *testing.T) {
	ctx := context.Background()
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("u1", "k1", []byte("t1")))
	ctx = keys.ContextWithKeyStore(ctx, ks)

	tok, ok := keys.TokenFromContext(ctx, "u1", "k1")
	if !ok {
		t.Fatal("expected token")
	}
	if got, want := string(tok.Value()), "t1"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	_, ok = keys.TokenFromContext(ctx, "u1", "missing")
	if ok {
		t.Error("expected no token")
	}

	ctxNoStore := context.Background()
	_, ok = keys.TokenFromContext(ctxNoStore, "u1", "k1")
	if ok {
		t.Error("expected no token")
	}
}

// TestKeyInfoFromContext covers the lookup rules documented on
// KeyInfoFromContext: an id is required, a specified user is looked up
// exactly, and an unspecified user falls back to a lookup by id alone,
// which only succeeds if the id is unique across all users.
func TestKeyInfoFromContext(t *testing.T) {
	store := keys.NewInMemoryKeyStore()
	store.Add(keys.NewInfo("user1", "key1", []byte("t1")))
	store.Add(keys.NewInfo("user2", "key2", []byte("t2")))
	// key3 is held by two users, so a lookup by id alone is ambiguous.
	store.Add(keys.NewInfo("user1", "key3", []byte("t3a")))
	store.Add(keys.NewInfo("user2", "key3", []byte("t3b")))
	ctx := keys.ContextWithKeyStore(context.Background(), store)

	t.Run("exact user and id", func(t *testing.T) {
		info, ok := keys.KeyInfoFromContext(ctx, "user1", "key1")
		if !ok {
			t.Fatal("expected to find key1/user1")
		}
		if got, want := string(info.Token().Value()), "t1"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("exact user and id not found", func(t *testing.T) {
		if _, ok := keys.KeyInfoFromContext(ctx, "user1", "key2"); ok {
			t.Error("key2 belongs to user2, want not found")
		}
	})

	t.Run("no user falls back to a unique id lookup", func(t *testing.T) {
		info, ok := keys.KeyInfoFromContext(ctx, "", "key2")
		if !ok {
			t.Fatal("expected to find the unique key2")
		}
		if got, want := string(info.Token().Value()), "t2"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("no user and an ambiguous id fails", func(t *testing.T) {
		if _, ok := keys.KeyInfoFromContext(ctx, "", "key3"); ok {
			t.Error("key3 is held by two users, want not found")
		}
		// The key remains reachable when the user is specified.
		if _, ok := keys.KeyInfoFromContext(ctx, "user1", "key3"); !ok {
			t.Error("key3/user1: expected to find the key")
		}
	})

	t.Run("empty id always fails", func(t *testing.T) {
		if _, ok := keys.KeyInfoFromContext(ctx, "user1", ""); ok {
			t.Error("empty id with a user: want not found")
		}
		if _, ok := keys.KeyInfoFromContext(ctx, "", ""); ok {
			t.Error("empty id and no user: want not found")
		}
		// Even a key stored with an empty id is not returned.
		store.Add(keys.NewInfo("user1", "", []byte("t-empty")))
		if _, ok := keys.KeyInfoFromContext(ctx, "user1", ""); ok {
			t.Error("empty id: want not found even when such a key exists")
		}
		store.Delete("user1", "")
	})

	t.Run("no key store in context", func(t *testing.T) {
		if _, ok := keys.KeyInfoFromContext(context.Background(), "user1", "key1"); ok {
			t.Error("no store in context: want not found")
		}
	})

	t.Run("key store explicitly removed from context", func(t *testing.T) {
		removed := keys.ContextWithoutKeyStore(ctx)
		if _, ok := keys.KeyInfoFromContext(removed, "user1", "key1"); ok {
			t.Error("store removed from context: want not found")
		}
	})
}

// TestKeyInfosFromContext covers the multi-key lookup performed by
// KeyInfosFromContext: each spec is resolved independently via
// KeyInfoFromContext, so the same exact/unique/ambiguous rules apply to each
// one, and the first spec that cannot be resolved fails the whole call.
func newKeyInfosTestContext() context.Context {
	store := keys.NewInMemoryKeyStore()
	store.Add(keys.NewInfo("user1", "key1", []byte("t1")))
	store.Add(keys.NewInfo("user2", "key2", []byte("t2")))
	// key3 is held by two users, so a lookup by id alone is ambiguous.
	store.Add(keys.NewInfo("user1", "key3", []byte("t3a")))
	store.Add(keys.NewInfo("user2", "key3", []byte("t3b")))
	return keys.ContextWithKeyStore(context.Background(), store)
}

func TestKeyInfosFromContext(t *testing.T) {
	ctx := newKeyInfosTestContext()

	infos, err := keys.KeyInfosFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infos == nil {
		t.Error("got a nil slice, want a non-nil empty one")
	}
	if len(infos) != 0 {
		t.Errorf("got %d infos, want 0", len(infos))
	}
}

// TestKeyInfosFromContextResolution covers that each spec is resolved by the
// same rules as KeyInfoFromContext: an exact user/id match, a fallback to a
// unique id when the user is unspecified, and the results kept in the order
// the specs were given.
func TestKeyInfosFromContextResolution(t *testing.T) {
	ctx := newKeyInfosTestContext()

	infos, err := keys.KeyInfosFromContext(ctx,
		keys.KeySpec{User: "user2", ID: "key3"},
		keys.KeySpec{User: "user1", ID: "key1"},
		keys.KeySpec{ID: "key2"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("got %d infos, want 3", len(infos))
	}
	got := []string{
		string(infos[0].Token().Value()),
		string(infos[1].Token().Value()),
		string(infos[2].Token().Value()),
	}
	want := []string{"t3b", "t1", "t2"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func testKeyInfosAmbiguous(ctx context.Context, t *testing.T) {
	infos, err := keys.KeyInfosFromContext(ctx,
		keys.KeySpec{User: "user1", ID: "key1"},
		keys.KeySpec{ID: "key3"},
	)
	if err == nil {
		t.Fatal("expected an error for the ambiguous key3 lookup")
	}
	if infos != nil {
		t.Errorf("got %v, want nil infos on error", infos)
	}
	if got, want := err.Error(), `ambiguous key "key3": multiple users match this id`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testKeyInfosUnresolvableStops(ctx context.Context, t *testing.T) {
	// key2 belongs to user2 here, not user1, so this must fail before
	// key1 -- which resolves fine on its own -- is ever reached, since
	// KeyInfosFromContext must not partially succeed.
	_, err := keys.KeyInfosFromContext(ctx,
		keys.KeySpec{User: "user1", ID: "key2"},
		keys.KeySpec{User: "user1", ID: "key1"},
	)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), `key not found for user "user1" and id "key2"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testKeyInfosNotFound(ctx context.Context, t *testing.T) {
	_, err := keys.KeyInfosFromContext(ctx, keys.KeySpec{ID: "missing"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), `key "missing" not found`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testKeyInfosEmptyID(ctx context.Context, t *testing.T) {
	_, err := keys.KeyInfosFromContext(ctx, keys.KeySpec{ID: ""})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "empty key id"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	_, err = keys.KeyInfosFromContext(ctx, keys.KeySpec{User: "user1", ID: ""})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), `empty key id for user "user1"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testKeyInfosNoStore(ctx context.Context, t *testing.T) {
	_, err := keys.KeyInfosFromContext(context.Background(), keys.KeySpec{User: "user1", ID: "key1"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "no key store in context"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Also fails when 0 specs are provided.
	_, err = keys.KeyInfosFromContext(context.Background())
	if err == nil {
		t.Fatal("expected an error with 0 specs and no store")
	}
	if got, want := err.Error(), "no key store in context"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	removed := keys.ContextWithoutKeyStore(ctx)
	_, err = keys.KeyInfosFromContext(removed, keys.KeySpec{User: "user1", ID: "key1"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if got, want := err.Error(), "no key store in context"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func testKeyInfosCanceled(ctx context.Context, t *testing.T) {
	cancCtx, cancel := context.WithCancel(ctx)
	cancel()
	_, err := keys.KeyInfosFromContext(cancCtx, keys.KeySpec{User: "user1", ID: "key1"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

// TestKeyInfosFromContextErrors covers error cases in KeyInfosFromContext:
// ambiguous ids, keys not found, empty ids, missing key store, and canceled contexts.
func TestKeyInfosFromContextErrors(t *testing.T) {
	ctx := newKeyInfosTestContext()

	t.Run("an ambiguous id fails even with other resolvable specs", func(t *testing.T) {
		testKeyInfosAmbiguous(ctx, t)
	})
	t.Run("the first unresolvable spec stops the lookup", func(t *testing.T) {
		testKeyInfosUnresolvableStops(ctx, t)
	})
	t.Run("key not found by id alone", func(t *testing.T) {
		testKeyInfosNotFound(ctx, t)
	})
	t.Run("empty key id", func(t *testing.T) {
		testKeyInfosEmptyID(ctx, t)
	})
	t.Run("no key store in context", func(t *testing.T) {
		testKeyInfosNoStore(ctx, t)
	})
	t.Run("canceled context", func(t *testing.T) {
		testKeyInfosCanceled(ctx, t)
	})
}

func TestInMemoryKeyStoreGetSpecs(t *testing.T) {
	store := keys.NewInMemoryKeyStore()
	store.Add(keys.NewInfo("u1", "k1", []byte("t1")))
	store.Add(keys.NewInfo("u2", "k2", []byte("t2")))
	store.Add(keys.NewInfo("u1", "k3", []byte("t3a")))
	store.Add(keys.NewInfo("u2", "k3", []byte("t3b")))

	// Empty specs returns non-nil slice and nil error.
	infos, err := store.GetSpecs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infos == nil || len(infos) != 0 {
		t.Errorf("got %v, want non-nil empty slice", infos)
	}

	// Successful batch lookup.
	infos, err = store.GetSpecs(
		keys.KeySpec{User: "u1", ID: "k1"},
		keys.KeySpec{ID: "k2"},
		keys.KeySpec{User: "u2", ID: "k3"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 3 {
		t.Fatalf("got %d infos, want 3", len(infos))
	}

	// Error clearing: ensure partially matched results are discarded on error.
	infos, err = store.GetSpecs(
		keys.KeySpec{User: "u1", ID: "k1"},
		keys.KeySpec{ID: "k_nonexistent"},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if infos != nil {
		t.Errorf("got %v, want nil slice on error", infos)
	}
}

func TestInMemoryKeyStoreGetSpecsConcurrent(t *testing.T) {
	store := keys.NewInMemoryKeyStore()
	store.Add(keys.NewInfo("u1", "k1", []byte("t1")))
	store.Add(keys.NewInfo("u2", "k2", []byte("t2")))

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Concurrent reader.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			infos, err := store.GetSpecs(
				keys.KeySpec{User: "u1", ID: "k1"},
				keys.KeySpec{User: "u2", ID: "k2"},
			)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(infos) != 2 {
				t.Errorf("got %d infos, want 2", len(infos))
				return
			}
		}
	}()

	// Concurrent writer adding and deleting unrelated keys.
	wg.Add(1)
	go func() {
		defer wg.Done()
		var i int
		for ctx.Err() == nil {
			id := fmt.Sprintf("temp_%d", i)
			store.Add(keys.NewInfo("u_temp", id, []byte("val")))
			store.Delete("u_temp", id)
			i++
		}
	}()

	wg.Wait()
}
