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
	k1 := keys.NewInfo("ctx-key", "ctx-user", []byte("ctx-token"))
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
	k1 := keys.NewInfo("k1", "u1", []byte("t1"))
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
	k2 := keys.NewInfo("k2", "u2", []byte("t2"))
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

func TestTokenFromContext(t *testing.T) {
	ctx := context.Background()
	ks := keys.NewInMemoryKeyStore()
	ks.Add(keys.NewInfo("k1", "u1", []byte("t1")))
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
