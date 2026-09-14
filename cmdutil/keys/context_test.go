// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"testing"

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
