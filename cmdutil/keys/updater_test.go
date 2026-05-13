// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys_test

import (
	"context"
	"testing"
	"time"

	"cloudeng.io/cmdutil/keys"
)

func TestKeyStoreUpdaterYAML(t *testing.T) {
	ctx := t.Context()
	mfs := &mockFS{data: map[string][]byte{"keys.yaml": []byte(yamlList)}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshYAML(ctx, mfs, "keys.yaml")
	defer u.Stop(ctx)

	// Trigger a direct read to verify the store without waiting for the ticker.
	if err := ims.ReadYAML(ctx, mfs, "keys.yaml"); err != nil {
		t.Fatalf("ReadYAML: %v", err)
	}

	if _, ok := ims.Get("user1", "key1"); !ok {
		t.Error("key1 not found after YAML read")
	}
	if _, ok := ims.Get("user2", "key2"); !ok {
		t.Error("key2 not found after YAML read")
	}
}

func TestKeyStoreUpdaterJSON(t *testing.T) {
	ctx := t.Context()
	mfs := &mockFS{data: map[string][]byte{"keys.json": []byte(jsonList)}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshJSON(ctx, mfs, "keys.json")
	defer u.Stop(ctx)

	if err := ims.ReadJSON(ctx, mfs, "keys.json"); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	if _, ok := ims.Get("user1", "key1"); !ok {
		t.Error("key1 not found after JSON read")
	}
	if _, ok := ims.Get("user2", "key2"); !ok {
		t.Error("key2 not found after JSON read")
	}
}

func TestKeyStoreUpdaterMultipleFiles(t *testing.T) {
	ctx := t.Context()
	mfs := &mockFS{data: map[string][]byte{
		"a.yaml": []byte(yamlList),
		"b.yaml": []byte(yamlListExtra),
	}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshYAML(ctx, mfs, "a.yaml", "b.yaml")
	defer u.Stop(ctx)

	if err := ims.ReadYAML(ctx, mfs, "a.yaml"); err != nil {
		t.Fatalf("ReadYAML a.yaml: %v", err)
	}
	if err := ims.ReadYAML(ctx, mfs, "b.yaml"); err != nil {
		t.Fatalf("ReadYAML b.yaml: %v", err)
	}

	if _, ok := ims.Get("user1", "key1"); !ok {
		t.Error("key1 not found")
	}
	if _, ok := ims.Get("user2", "key2"); !ok {
		t.Error("key2 not found")
	}
}

func TestKeyStoreUpdaterMixedSchedule(t *testing.T) {
	ctx := t.Context()
	mfs := &mockFS{data: map[string][]byte{
		"keys.yaml": []byte(yamlList),
		"keys.json": []byte(jsonMap),
	}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshYAML(ctx, mfs, "keys.yaml")
	u.ScheduleRefreshJSON(ctx, mfs, "keys.json")
	defer u.Stop(ctx)

	if err := ims.ReadYAML(ctx, mfs, "keys.yaml"); err != nil {
		t.Fatalf("ReadYAML: %v", err)
	}
	if err := ims.ReadJSON(ctx, mfs, "keys.json"); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}

	if ims.Len() == 0 {
		t.Error("expected non-empty store after mixed reads")
	}
}

func TestKeyStoreUpdaterStop(t *testing.T) {
	ctx := t.Context()
	mfs := &mockFS{data: map[string][]byte{"keys.yaml": []byte(yamlList)}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshYAML(ctx, mfs, "keys.yaml")

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	u.Stop(stopCtx) // must return without hanging
	u.Stop(stopCtx) // second call must be a no-op
}

func TestKeyStoreUpdaterContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	mfs := &mockFS{data: map[string][]byte{"keys.yaml": []byte(yamlList)}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshYAML(ctx, mfs, "keys.yaml")

	cancel() // goroutine exits via ctx.Done()

	stopCtx, stopCancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer stopCancel()
	u.Stop(stopCtx) // must return promptly
}

func TestKeyStoreUpdaterRefreshError(t *testing.T) {
	ctx := t.Context()
	mfs := &mockFS{data: map[string][]byte{}}

	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)
	u.ScheduleRefreshYAML(ctx, mfs, "missing.yaml")
	defer u.Stop(ctx)

	// Direct read must fail and leave the store empty.
	if err := ims.ReadYAML(ctx, mfs, "missing.yaml"); err == nil {
		t.Error("expected error reading missing file, got nil")
	}
	if ims.Len() != 0 {
		t.Errorf("expected empty store after read errors, got %d keys", ims.Len())
	}
}

func TestKeyStoreUpdaterNoSchedule(t *testing.T) {
	ims := keys.NewInMemoryKeyStore()
	u := keys.NewKeyStoreUpdater(ims)

	stopCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	u.Stop(stopCtx) // nothing scheduled; must be a clean no-op
}
