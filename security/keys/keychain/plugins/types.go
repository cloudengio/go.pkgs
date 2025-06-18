// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package plugins

// Request represents the request to the keychain plugin.
type Request struct {
	Account  string `json:"account"`
	Keyname  string `json:"keyname"`
	WriteKey bool   `json:"write_key,omitempty"` // if true, write the key
	Contents string `json:"contents,omitempty"`  // base64 encoded contents for writing
}

// Response represents the response from the keychain plugin.
type Response struct {
	Account  string `json:"account"`
	Keyname  string `json:"keyname"`
	Contents string `json:"contents"` // base64 encoded
	Error    string `json:"error,omitempty"`
}
