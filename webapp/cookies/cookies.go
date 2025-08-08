// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cookies

import (
	"net/http"
	"time"
)

// T represents a named cookie. It is primarily intended to document
// and track the use of cookies in a web application.
type T string

func setCookie(rw http.ResponseWriter, name, value string, expires time.Duration) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(expires), // Set a suitable expiration
		HttpOnly: true,                    // Essential for security
		Secure:   true,                    // Essential for HTTPS
		SameSite: http.SameSiteStrictMode, // Recommended for CSRF protection
	}
	http.SetCookie(rw, cookie)
}

func readAndClearCookie(rw http.ResponseWriter, r *http.Request, name string) (string, bool) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	val := cookie.Value
	cookie.MaxAge = -1
	cookie.Value = ""
	http.SetCookie(rw, cookie)
	return val, true
}

// SetSecureWithExpiration sets a cookie with a specific expiration time.
// The path is set to "/" to make the cookie available across the entire site,
// and the cookie is marked as secure and HTTP-only and SameSiteStrictMode.
func (c T) SetSecureWithExpiration(rw http.ResponseWriter, value string, expires time.Duration) {
	setCookie(rw, string(c), value, expires)
}

// Set sets the supplied cookie with the name of the cookie specified
// in the receiver but using all other values from the supplied cookie.
func (c T) Set(rw http.ResponseWriter, ck *http.Cookie) {
	ck.Name = string(c)
	http.SetCookie(rw, ck)
}

// ReadAndClear reads a cookie and requests its removal by setting
// its MaxAge to -1 and its value to an empty string.
func (c T) ReadAndClear(rw http.ResponseWriter, r *http.Request) (string, bool) {
	return readAndClearCookie(rw, r, string(c))
}

// Read reads the cookie from the request and returns its value.
// If the cookie is not present, it returns an empty string and false.
func (c T) Read(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(string(c))
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}
