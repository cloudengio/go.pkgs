// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keyscmd

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/cmdutil/keys"
)

// NewSecret generates a new random secret of the specified size in bytes.
func newSecret(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("size must be > 0")
	}
	raw := make([]byte, size)
	n, err := rand.Read(raw)
	if err != nil {
		return nil, fmt.Errorf("generating random secret: %v", err)
	}
	if n != size {
		return nil, fmt.Errorf("generated secret of incorrect size: expected %d bytes, got %d bytes", size, n)
	}
	return raw, nil
}

// SecretFormat represents the format in which the secret is represented, such
// as raw bytes, hexadecimal, or base64 encoding.
type SecretFormat int

const (
	// SecretFormatRaw indicates that the secret is stored in raw byte format.
	SecretFormatRaw SecretFormat = iota
	// SecretFormatHex indicates that the secret is stored in hexadecimal format.
	SecretFormatHex
	// SecretFormatBase64 indicates that the secret is stored in base64.StdEncoding format.
	SecretFormatBase64
)

// EnumValues implements flags.EnumValues for SecretFormat. A new map is
// returned on each call so that callers cannot mutate the values shared by
// every SecretFormat.
func (SecretFormat) EnumValues() map[string]SecretFormat {
	return map[string]SecretFormat{
		"raw":    SecretFormatRaw,
		"hex":    SecretFormatHex,
		"base64": SecretFormatBase64,
	}
}

type SecretConfigFlags struct {
	Size   int                      `subcmd:"size,32,size of the secret in bytes"`
	Format flags.Enum[SecretFormat] `subcmd:"format,hex,'format of the secret, one of raw, hex, base64'"`
	ID     string                   `subcmd:"id,,id of the key"`
	User   string                   `subcmd:"user,,user/owner associated with the key"`
}

func (sf SecretConfigFlags) SecretConfig() SecretConfig {
	return SecretConfig{
		Size:   sf.Size,
		Format: sf.Format.Value,
		ID:     sf.ID,
		User:   sf.User,
	}
}

type SecretConfig struct {
	Size   int          `yaml:"key-size" doc:"size of the secret in bytes"`
	Format SecretFormat `yaml:"key-format" doc:"format of the secret, one of raw, hex, base64"`
	ID     string       `yaml:"key-id" doc:"id of the key"`
	User   string       `yaml:"key-user" doc:"user/owner associated with the key"`
}

// NewSecret generates a new random secret of the specified size in bytes and
// format and returns it as a keys.Info object.
func (sc SecretConfig) New() (keys.Info, error) {
	if sc.Size <= 0 {
		return keys.Info{}, fmt.Errorf("invalid secret size %d", sc.Size)
	}
	raw, err := newSecret(sc.Size)
	if err != nil {
		return keys.Info{}, err
	}
	switch sc.Format {
	case SecretFormatRaw:
		return keys.NewInfo(sc.ID, sc.User, raw), nil
	case SecretFormatHex:
		encoded := make([]byte, hex.EncodedLen(len(raw)))
		hex.Encode(encoded, raw)
		return keys.NewInfo(sc.ID, sc.User, encoded), nil
	case SecretFormatBase64:
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
		base64.StdEncoding.Encode(encoded, raw)
		return keys.NewInfo(sc.ID, sc.User, encoded), nil
	default:
		return keys.Info{}, fmt.Errorf("unsupported secret format: %v", sc.Format)
	}
}
