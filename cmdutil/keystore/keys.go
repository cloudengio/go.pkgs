package keystore

import (
	"context"
	"fmt"
	"io/fs"

	"cloudeng.io/cmdutil/cmdyaml"
)

// KeyInfo represents a specific key configuration and is intended
// to be reused and referred to by it's key_id.
type KeyInfo struct {
	ID    string `yaml:"key_id"`
	User  string `yaml:"user"`
	Token string `yaml:"token"`
}

// Keys is a map of ID/key_id to KeyInfo
type Keys map[string]KeyInfo

func (k KeyInfo) String() string {
	return k.ID + "[" + k.User + "]	"
}

// ParseFile parses an auth file into an AuthInfo map and stores that
// in the returned context. The file may be stored in any
func ParseFile(fs fs.ReadFileFS, filename string) (Keys, error) {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse parses the supplied data into an AuthInfo map.
func Parse(data []byte) (Keys, error) {
	var auth []KeyInfo
	if err := cmdyaml.ParseConfig(data, &auth); err != nil {
		return nil, err
	}
	am := Keys{}
	for _, a := range auth {
		if a.ID == "" {
			return nil, fmt.Errorf("key_id is required")
		}
		if a.User == "" {
			return nil, fmt.Errorf("user is required")
		}
		if a.Token == "" {
			return nil, fmt.Errorf("token is required")
		}
		am[a.ID] = a
	}
	return am, nil
}

type ctxKey struct{}

func ContextWithAuth(ctx context.Context, am Keys) context.Context {
	return context.WithValue(ctx, ctxKey{}, am)
}

func AuthFromContextForID(ctx context.Context, id string) KeyInfo {
	am, ok := ctx.Value(ctxKey{}).(Keys)
	if !ok {
		return KeyInfo{}
	}
	return am[id]
}
