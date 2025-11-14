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
	testCookieName   cookies.T      = "my-test-cookie"
	secureCookieName cookies.Secure = "my-secure-cookie"
	testValue        string         = "hello-world"
)

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

func TestSecure_Set(t *testing.T) {
	rec := httptest.NewRecorder()
	ck := &http.Cookie{
		Value:  testValue,
		Domain: "example.com",
		Path:   "/",
	}

	secureCookieName.Set(rec, ck)

	res := rec.Result()
	defer res.Body.Close()

	if len(res.Cookies()) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(res.Cookies()))
	}

	cookie := res.Cookies()[0]

	if got, want := cookie.Name, string(secureCookieName); got != want {
		t.Errorf("got cookie name %q, want %q", got, want)
	}
	if got, want := cookie.Value, testValue; got != want {
		t.Errorf("got cookie value %q, want %q", got, want)
	}
	if got, want := cookie.Domain, "example.com"; got != want {
		t.Errorf("got cookie domain %q, want %q", got, want)
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
}

func TestSecure_Read(t *testing.T) {
	// Case 1: Cookie exists.
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: string(secureCookieName), Value: testValue})

	val, ok := secureCookieName.Read(req)
	if !ok {
		t.Error("expected to find cookie, but did not")
	}
	if val != testValue {
		t.Errorf("got value %q, want %q", val, testValue)
	}

	// Case 2: Cookie does not exist.
	reqNoCookie := httptest.NewRequest("GET", "/", nil)
	val, ok = secureCookieName.Read(reqNoCookie)
	if ok {
		t.Error("did not expect to find cookie, but did")
	}
	if val != "" {
		t.Errorf("got value %q, want empty string", val)
	}
}

func TestSecure_ReadAndClear(t *testing.T) {
	// Case 1: Cookie exists.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: string(secureCookieName), Value: testValue})

	val, ok := secureCookieName.ReadAndClear(rec, req)
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
	val, ok = secureCookieName.ReadAndClear(recNoCookie, reqNoCookie)
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

func TestScopeAndDuration_SetDefaults(t *testing.T) {
	t.Run("Use Defaults", func(t *testing.T) {
		var sd cookies.ScopeAndDuration // Empty

		result := sd.SetDefaults("default-domain.com", "/default-path", 30*time.Minute)

		if result.Domain != "default-domain.com" {
			t.Errorf("Domain not set to default: got %q, want %q", result.Domain, "default-domain.com")
		}
		if result.Path != "/default-path" {
			t.Errorf("Path not set to default: got %q, want %q", result.Path, "/default-path")
		}
		if result.Duration != 30*time.Minute {
			t.Errorf("Duration not set to default: got %v, want %v", result.Duration, 30*time.Minute)
		}
	})

	t.Run("Keep Existing Values", func(t *testing.T) {
		sd := cookies.ScopeAndDuration{
			Domain:   "original-domain.com",
			Path:     "/original-path",
			Duration: 15 * time.Minute,
		}

		result := sd.SetDefaults("default-domain.com", "/default-path", 30*time.Minute)

		if result.Domain != "original-domain.com" {
			t.Errorf("Domain was overridden: got %q, want %q", result.Domain, "original-domain.com")
		}
		if result.Path != "/original-path" {
			t.Errorf("Path was overridden: got %q, want %q", result.Path, "/original-path")
		}
		if result.Duration != 15*time.Minute {
			t.Errorf("Duration was overridden: got %v, want %v", result.Duration, 15*time.Minute)
		}
	})
}

func TestScopeAndDuration_Cookie(t *testing.T) {
	sd := cookies.ScopeAndDuration{
		Domain:   "example.com",
		Path:     "/api",
		Duration: 20 * time.Minute,
	}

	cookie := sd.Cookie(testValue)

	if cookie.Domain != "example.com" {
		t.Errorf("Cookie domain: got %q, want %q", cookie.Domain, "example.com")
	}
	if cookie.Path != "/api" {
		t.Errorf("Cookie path: got %q, want %q", cookie.Path, "/api")
	}
	if cookie.Value != testValue {
		t.Errorf("Cookie value: got %q, want %q", cookie.Value, testValue)
	}

	// The exact expiration time depends on when the test runs, so just check that
	// it's between now and now+Duration
	if time.Until(cookie.Expires) > sd.Duration {
		t.Errorf("cookie expiration is too far in the future")
	}
	// Allow for a small delta for test execution time.
	if time.Until(cookie.Expires) < sd.Duration-time.Second {
		t.Errorf("cookie expiration is too soon, got %v, want > %v", time.Until(cookie.Expires), sd.Duration-time.Second)
	}
}
