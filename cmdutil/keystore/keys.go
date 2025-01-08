package keystore

import (
	"context"
	"fmt"

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

// ParseConfigFile calls cmdyaml.ParseConfigFile for Keys.
func ParseConfigFile(ctx context.Context, filename string) (Keys, error) {
	var auth []KeyInfo
	if err := cmdyaml.ParseConfigFile(ctx, filename, &auth); err != nil {
		return nil, err
	}
	return newKeys(auth)
}

// ParseConfigURI calls cmdyaml.ParseConfigURI for Keys.
func ParseConfigURI(ctx context.Context, filename string, handlers map[string]cmdyaml.URLHandler) (Keys, error) {
	var auth []KeyInfo
	if err := cmdyaml.ParseConfigURI(ctx, filename, &auth, handlers); err != nil {
		return nil, err
	}
	return newKeys(auth)
}

func newKeys(auth []KeyInfo) (Keys, error) {
	am := Keys{}
	for _, a := range auth {
		if a.ID == "" {
			return nil, fmt.Errorf("key_id is required")
		}
		if a.User == "" {
			return nil, fmt.Errorf("user is required for key_id: %v", a.ID)
		}
		if a.Token == "" {
			return nil, fmt.Errorf("token is required for key_id: %v", a.ID)
		}
		am[a.ID] = a
	}
	return am, nil
}

// Parse parses the supplied data into an AuthInfo map.
func Parse(data []byte) (Keys, error) {
	var auth []KeyInfo
	if err := cmdyaml.ParseConfig(data, &auth); err != nil {
		return nil, err
	}
	return newKeys(auth)
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
