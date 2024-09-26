package keystore_test

import (
	"context"
	"io/fs"
	"reflect"
	"testing"

	"cloudeng.io/cmdutil/keystore"
)

type rfs struct{}

func (rfs) Open(_ string) (fs.File, error) {
	return nil, nil
}

func (rfs) ReadFile(_ string) ([]byte, error) {
	return []byte(`- key_id: "123"
  user: user1
  token: token1
- key_id: "456"
  user: user2
  token: token2
`), nil
}

func TestParse(t *testing.T) {
	am, err := keystore.ParseFile(&rfs{}, "filename")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := am, (keystore.Keys{
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
	ai := keystore.Keys{
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
	ctx := keystore.ContextWithAuth(context.Background(), ai)
	var empty keystore.KeyInfo
	if got, want := keystore.AuthFromContextForID(ctx, "2356"), empty; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
