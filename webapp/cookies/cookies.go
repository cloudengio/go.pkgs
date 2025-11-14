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

// Secure represents a named cookie that is set 'securely'.
// It is primarily intended to document and track the use of cookies in a
// web application.
type Secure string

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

// Set sets the supplied cookie with the name of the cookie specified
// in the receiver. It overwrites the Name in ck.
// All other fields in ck are used as specified.
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

// Set sets the supplied cookie securely with the name of the cookie specified
// in the receiver and secure values for
// HttpOnly, Secure and SameSite (true, true, SameSiteStrictMode).
// All other fields in ck are used as specified.
func (c Secure) Set(rw http.ResponseWriter, ck *http.Cookie) {
	ck.Name = string(c)
	ck.HttpOnly = true
	ck.Secure = true
	ck.SameSite = http.SameSiteStrictMode
	http.SetCookie(rw, ck)
}

// ReadAndClear reads a cookie and requests its removal by setting
// its MaxAge to -1 and its value to an empty string.
func (c Secure) ReadAndClear(rw http.ResponseWriter, r *http.Request) (string, bool) {
	return readAndClearCookie(rw, r, string(c))
}

// Read reads the cookie from the request and returns its value.
// If the cookie is not present, it returns an empty string and false.
func (c Secure) Read(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(string(c))
	if err != nil {
		return "", false
	}
	return cookie.Value, true
}

// ScopeAndDuration represents the scope and duration settings for cookies.
type ScopeAndDuration struct {
	Domain   string
	Path     string
	Duration time.Duration
}

// SetDefaults uses the supplied values as defaults for ScopeAndDuration if the
// current values are not already set.
func (d ScopeAndDuration) SetDefaults(domain, path string, duration time.Duration) ScopeAndDuration {
	if d.Domain == "" {
		d.Domain = domain
	}
	if d.Path == "" {
		d.Path = path
	}
	if d.Duration == 0 {
		d.Duration = duration
	}
	return d
}

// Cookie returns a new http.Cookie with the specified value and the
// scope and duration settings from the ScopeAndDuration receiver.
func (d ScopeAndDuration) Cookie(value string) *http.Cookie {
	return &http.Cookie{
		Domain:  d.Domain,
		Path:    d.Path,
		Expires: time.Now().Add(d.Duration),
		Value:   value,
	}
}
