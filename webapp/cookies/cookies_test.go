// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cookies_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloudeng.io/webapp/cookies"
)

const (
	testCookieName cookies.T = "my-test-cookie"
	testValue      string    = "hello-world"
)

func TestSetSecureWithExpiration(t *testing.T) {
	rec := httptest.NewRecorder()
	expires := 10 * time.Minute

	testCookieName.SetSecureWithExpiration(rec, testValue, expires)

	res := rec.Result()
	defer res.Body.Close()

	if len(res.Cookies()) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(res.Cookies()))
	}

	cookie := res.Cookies()[0]

	if got, want := cookie.Name, string(testCookieName); got != want {
		t.Errorf("got cookie name %q, want %q", got, want)
	}
	if got, want := cookie.Value, testValue; got != want {
		t.Errorf("got cookie value %q, want %q", got, want)
	}
	if got, want := cookie.Path, "/"; got != want {
		t.Errorf("got cookie path %q, want %q", got, want)
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}
	if !cookie.Secure {
		t.Error("expected Secure to be true")
	}
	if got, want := cookie.SameSite, http.SameSiteStrictMode; got != want {
		t.Errorf("got SameSite mode %v, want %v", got, want)
	}

	// Check that the expiration is in the future and close to what we set.
	if time.Until(cookie.Expires) > expires {
		t.Errorf("cookie expiration is too far in the future")
	}
	if time.Until(cookie.Expires) < expires-time.Minute {
		t.Errorf("cookie expiration is too soon")
	}
}

func TestSet(t *testing.T) {
	rec := httptest.NewRecorder()
	customCookie := &http.Cookie{
		Name:     "original-name", // This should be overridden.
		Value:    "custom-value",
		Path:     "/custom",
		SameSite: http.SameSiteLaxMode,
	}

	testCookieName.Set(rec, customCookie)

	res := rec.Result()
	defer res.Body.Close()

	if len(res.Cookies()) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(res.Cookies()))
	}

	cookie := res.Cookies()[0]

	if got, want := cookie.Name, string(testCookieName); got != want {
		t.Errorf("got cookie name %q, want %q (should be overridden)", got, want)
	}
	if got, want := cookie.Value, customCookie.Value; got != want {
		t.Errorf("got cookie value %q, want %q", got, want)
	}
	if got, want := cookie.Path, customCookie.Path; got != want {
		t.Errorf("got cookie path %q, want %q", got, want)
	}
	if got, want := cookie.SameSite, customCookie.SameSite; got != want {
		t.Errorf("got SameSite mode %v, want %v", got, want)
	}
}

func TestRead(t *testing.T) {
	// Case 1: Cookie exists.
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: string(testCookieName), Value: testValue})

	val, ok := testCookieName.Read(req)
	if !ok {
		t.Error("expected to find cookie, but did not")
	}
	if val != testValue {
		t.Errorf("got value %q, want %q", val, testValue)
	}

	// Case 2: Cookie does not exist.
	reqNoCookie := httptest.NewRequest("GET", "/", nil)
	val, ok = testCookieName.Read(reqNoCookie)
	if ok {
		t.Error("did not expect to find cookie, but did")
	}
	if val != "" {
		t.Errorf("got value %q, want empty string", val)
	}
}

func TestReadAndClear(t *testing.T) {
	// Case 1: Cookie exists.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: string(testCookieName), Value: testValue})

	val, ok := testCookieName.ReadAndClear(rec, req)
	if !ok {
		t.Error("expected to find cookie, but did not")
	}
	if val != testValue {
		t.Errorf("got value %q, want %q", val, testValue)
	}

	// Check that a clearing cookie was set in the response.
	res := rec.Result()
	defer res.Body.Close()
	if len(res.Cookies()) != 1 {
		t.Fatalf("expected 1 clearing cookie to be set, got %d", len(res.Cookies()))
	}
	clearingCookie := res.Cookies()[0]
	if clearingCookie.MaxAge != -1 {
		t.Errorf("expected MaxAge of -1 to clear cookie, got %d", clearingCookie.MaxAge)
	}
	if clearingCookie.Value != "" {
		t.Errorf("expected empty value to clear cookie, got %q", clearingCookie.Value)
	}

	// Case 2: Cookie does not exist.
	recNoCookie := httptest.NewRecorder()
	reqNoCookie := httptest.NewRequest("GET", "/", nil)
	val, ok = testCookieName.ReadAndClear(recNoCookie, reqNoCookie)
	if ok {
		t.Error("did not expect to find cookie, but did")
	}
	if val != "" {
		t.Errorf("got value %q, want empty string", val)
	}
	resNoCookie := recNoCookie.Result()
	defer resNoCookie.Body.Close()
	if len(resNoCookie.Cookies()) != 0 {
		t.Errorf("expected no cookies to be set when clearing a non-existent cookie, but got %d", len(resNoCookie.Cookies()))
	}
}
