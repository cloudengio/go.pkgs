// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !darwin

package keychain

import "errors"

func WriteSecureNote(_, _ string, _ []byte) error {
	return errors.New("not implemented on this platform")
}

func ReadSecureNote(_, _ string) ([]byte, error) {
	return nil, errors.New("not implemented on this platform")
}
