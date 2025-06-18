// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package unsafekeystore_test

import (
	"context"
	"io/fs"
	"reflect"
	"testing"

	"cloudeng.io/cmdutil/unsafekeystore"
	"cloudeng.io/file"
)

type rfs struct{}

func (rfs) Open(string) (fs.File, error) {
	return nil, nil
}

func (rfs) OpenCtx(_ context.Context, filename string) (fs.File, error) {
	return nil, nil
}

func (r rfs) ReadFileCtx(_ context.Context, filename string) ([]byte, error) {
	return r.ReadFile(filename)
}

func (rfs) ReadFile(string) ([]byte, error) {
	return []byte(`- key_id: "123"
  user: user1
  token: token1
- key_id: "456"
  user: user2
  token: token2
`), nil
}

func TestParse(t *testing.T) {
	ctx := context.Background()
	ctx = file.ContextWithFS(ctx, &rfs{})
	am, err := unsafekeystore.ParseConfigFile(ctx, "filename")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := am, (unsafekeystore.Keys{
		"123": {
			ID:    "123",
			User:  "user1",
			Token: "token1",
		},
		"456": {
			ID:    "456",
			User:  "user2",
			Token: "token2",
		}}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestKeysContext(t *testing.T) {
	ai := unsafekeystore.Keys{
		"123": {
			ID:    "123",
			User:  "user1",
			Token: "token1",
		},
		"456": {
			ID:    "456",
			User:  "user2",
			Token: "token2",
		},
	}
	ctx := unsafekeystore.ContextWithAuth(context.Background(), ai)
	var empty unsafekeystore.KeyInfo
	if got, want := unsafekeystore.AuthFromContextForID(ctx, "2356"), empty; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
