// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys

import "context"

type ctxKey struct{}

// ContextWithKeyStore returns a new context with the provided InMemoryKeyStore.
func ContextWithKeyStore(ctx context.Context, ims *InMemoryKeyStore) context.Context {
	return context.WithValue(ctx, ctxKey{}, ims)
}

// KeyStoreFromContext retrieves the InMemoryKeyStore from the context.
func KeyStoreFromContext(ctx context.Context) (*InMemoryKeyStore, bool) {
	am, ok := ctx.Value(ctxKey{}).(*InMemoryKeyStore)
	if !ok {
		return nil, false
	}
	return am, true
}

// ContextWithoutKeyStore returns a new context without an InMemoryKeyStore.
func ContextWithoutKeyStore(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, nil)
}

// KeyInfoFromContextForID retrieves the KeyInfo for the specified ID from the context.
func KeyInfoFromContextForID(ctx context.Context, id string) (Info, bool) {
	am, ok := ctx.Value(ctxKey{}).(*InMemoryKeyStore)
	if !ok {
		return Info{}, false
	}
	return am.Get(id)
}

// TokenFromContextForID retrieves the Token for the specified ID from the context.
func TokenFromContextForID(ctx context.Context, id string) (*Token, bool) {
	ki, ok := KeyInfoFromContextForID(ctx, id)
	if !ok {
		return nil, false
	}
	return ki.Token(), true
}

// ContextWithKey returns a new context with the provided KeyInfo added
// to an InMemoryKeyStore. If no InMemoryKeyStore exists in the context,
// a new one is created.
func ContextWithKey(ctx context.Context, ki Info) context.Context {
	ims, ok := KeyStoreFromContext(ctx)
	if !ok {
		ims = NewInMemoryKeyStore()
		ctx = ContextWithKeyStore(ctx, ims)
	}
	ims.Add(ki)
	return ctx
}
