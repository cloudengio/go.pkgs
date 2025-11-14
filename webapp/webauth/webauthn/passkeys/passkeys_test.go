// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	_ passkeys.WebAuthn     = (*mockWebAuthn)(nil)
	_ passkeys.LoginManager = (*mockLoginManager)(nil)
	_ passkeys.WebAuthn     = (*webauthn.WebAuthn)(nil)
	_ passkeys.LoginManager = (passkeys.LoginManager)(nil)
)

// mockWebAuthn is a simplified mock of webauthn.WebAuthn.
type mockWebAuthn struct{}

func (m *mockWebAuthn) BeginMediatedRegistration(user webauthn.User, _ protocol.CredentialMediationRequirement, _ ...webauthn.RegistrationOption) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	cred := &protocol.CredentialCreation{
		Response: protocol.PublicKeyCredentialCreationOptions{
			Challenge: protocol.URLEncodedBase64("test-challenge"),
			User: protocol.UserEntity{
				ID: user.WebAuthnID(),
				CredentialEntity: protocol.CredentialEntity{
					Name: user.WebAuthnName(),
				},
				DisplayName: user.WebAuthnDisplayName(),
			},
		},
	}
	sess := &webauthn.SessionData{
		Challenge: "dGVzdC1jaGFsbGVuZ2U", // "test-challenge" base64 encoded
		UserID:    user.WebAuthnID(),
	}
	return cred, sess, nil
}

func (m *mockWebAuthn) FinishRegistration(_ webauthn.User, _ webauthn.SessionData, _ *http.Request) (*webauthn.Credential, error) {
	return &webauthn.Credential{
		ID:              []byte("test-credential-id"),
		PublicKey:       []byte("test-public-key"),
		AttestationType: "none",
	}, nil
}

func (m *mockWebAuthn) BeginDiscoverableMediatedLogin(_ protocol.CredentialMediationRequirement, _ ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return &protocol.CredentialAssertion{
			Response: protocol.PublicKeyCredentialRequestOptions{
				Challenge: protocol.URLEncodedBase64("test-challenge"),
			},
		}, &webauthn.SessionData{
			Challenge: "dGVzdC1jaGFsbGVuZ2U",
		}, nil
}

func (m *mockWebAuthn) FinishPasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, _ *http.Request) (webauthn.User, *webauthn.Credential, error) {
	user, err := handler(nil, session.UserID)
	if err != nil {
		return nil, nil, err
	}
	cred := &webauthn.Credential{
		ID:        []byte("test-credential-id"),
		PublicKey: []byte("test-public-key"),
		Authenticator: webauthn.Authenticator{
			AAGUID: []byte("test-aaguid"),
		},
	}
	return user, cred, nil
}

// mockLoginManager implements the LoginManager interface for testing.
type mockLoginManager struct {
	authenticatedUserID passkeys.UserID
}

func (m *mockLoginManager) UserAuthenticated(_ http.ResponseWriter, userID passkeys.UserID) error {
	m.authenticatedUserID = userID
	return nil
}

func (m *mockLoginManager) AuthenticateUser(_ *http.Request) (passkeys.UserID, error) {
	return m.authenticatedUserID, nil
}

func TestRAMUserDatabase(t *testing.T) {
	db := passkeys.NewRAMUserDatabase()

	user, err := passkeys.NewUser("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("NewUser failed: %v", err)
	}

	// Test Store
	if err := db.Store(user); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Test Lookup
	retrievedUser, err := db.Lookup(user.ID())
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if !reflect.DeepEqual(retrievedUser, user) {
		t.Errorf("lookup returned different user data")
	}

	// Test session management
	sessionData := &webauthn.SessionData{
		Challenge: "test-challenge",
		UserID:    user.WebAuthnID(),
	}

	tmpKey, exists, err := db.Registering(user, sessionData)
	if err != nil {
		t.Fatalf("Registering failed: %v", err)
	}
	if exists {
		t.Error("user should not exist in session store yet")
	}
	if tmpKey == "" {
		t.Error("expected non-empty temporary key")
	}

	retrievedUser, retrievedSession, err := db.Registered(tmpKey)
	if err != nil {
		t.Fatalf("Registered failed: %v", err)
	}
	if !reflect.DeepEqual(retrievedUser, user) {
		t.Errorf("registered returned different user data")
	}
	if !reflect.DeepEqual(retrievedSession, sessionData) {
		t.Errorf("expected session %+v, got %+v", sessionData, retrievedSession)
	}
}

func TestHandler_BeginRegistration(t *testing.T) {
	mockWA := &mockWebAuthn{}
	db := passkeys.NewRAMUserDatabase()
	lm := &mockLoginManager{}
	handler := passkeys.NewHandler(mockWA, db, db, lm,
		passkeys.WithRegistrationOptions(
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				AuthenticatorAttachment: protocol.Platform,
				ResidentKey:             protocol.ResidentKeyRequirementRequired,
				UserVerification:        protocol.VerificationPreferred}),
		))

	reqBody := `{"email":"test@example.com","display_name":"Test User"}`
	req := httptest.NewRequest("POST", "/register/begin", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	handler.BeginRegistration(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp protocol.PublicKeyCredentialCreationOptions
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !bytes.Equal(resp.Challenge, protocol.URLEncodedBase64("test-challenge")) {
		t.Errorf("expected challenge %q, got %q", "test-challenge", resp.Challenge)
	}

	cookies := rec.Result().Cookies()
	var found bool
	for _, cookie := range cookies {
		if cookie.Name == string(passkeys.RegistrationCookie) {
			found = true
			if cookie.Value == "" {
				t.Error("registration cookie has empty value")
			}
			break
		}
	}
	if !found {
		t.Error("registration cookie not set")
	}
}

func TestHandler_FinishRegistration(t *testing.T) {
	mockWA := &mockWebAuthn{}
	db := passkeys.NewRAMUserDatabase()
	mw := &mockLoginManager{}
	handler := passkeys.NewHandler(mockWA, db, db, mw)

	user, err := passkeys.NewUser("test@example.com", "Test User")
	if err != nil {
		t.Fatal(err)
	}
	sessionData := &webauthn.SessionData{
		Challenge: "test-challenge",
		UserID:    user.WebAuthnID(),
	}
	tmpKey, _, _ := db.Registering(user, sessionData)

	req := httptest.NewRequest("POST", "/register/finish", nil)
	req.AddCookie(&http.Cookie{Name: string(passkeys.RegistrationCookie), Value: tmpKey})

	rec := httptest.NewRecorder()
	handler.FinishRegistration(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	storedUser, err := db.Lookup(user.ID())
	if err != nil {
		t.Fatalf("user not stored: %v", err)
	}
	if len(storedUser.WebAuthnCredentials()) != 1 {
		t.Errorf("expected 1 credential, got %d", len(storedUser.WebAuthnCredentials()))
	}
}

func TestHandler_FinishRegistration_MissingCookie(t *testing.T) {
	mockWA := &mockWebAuthn{}
	db := passkeys.NewRAMUserDatabase()
	mw := &mockLoginManager{}
	handler := passkeys.NewHandler(mockWA, db, db, mw)

	req := httptest.NewRequest("POST", "/register/finish", nil)
	rec := httptest.NewRecorder()

	handler.FinishRegistration(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var errResp struct{ Message string }
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Message != "missing registration cookie" {
		t.Errorf("expected error %q, got %q", "missing registration cookie", errResp.Message)
	}
}
