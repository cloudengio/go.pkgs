// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	"github.com/go-webauthn/webauthn/webauthn"
)

func TestNewUser(t *testing.T) {
	email := "test@example.com"
	displayName := "Test User"

	user, err := passkeys.NewUser(email, displayName)
	if err != nil {
		t.Fatalf("NewUser() error = %v, want nil", err)
	}
	if user == nil {
		t.Fatal("NewUser() returned a nil user")
	}

	if user.WebAuthnName() != email {
		t.Errorf("WebAuthnName() = %q, want %q", user.WebAuthnName(), email)
	}
	if user.WebAuthnDisplayName() != displayName {
		t.Errorf("WebAuthnDisplayName() = %q, want %q", user.WebAuthnDisplayName(), displayName)
	}

	id := user.ID()
	if id == nil {
		t.Fatal("ID() returned a nil UserID")
	}

	webAuthnID := user.WebAuthnID()
	if len(webAuthnID) != 64 {
		t.Errorf("len(WebAuthnID()) = %d, want 64", len(webAuthnID))
	}
	if bytes.Equal(webAuthnID, make([]byte, 64)) {
		t.Error("WebAuthnID() is all zeros, expected random bytes")
	}

	if creds := user.WebAuthnCredentials(); len(creds) != 0 {
		t.Errorf("WebAuthnCredentials() on new user should be empty, got %d credentials", len(creds))
	}
}

func TestUserID_Serialization(t *testing.T) {
	user, err := passkeys.NewUser("test@example.com", "Test User")
	if err != nil {
		t.Fatal(err)
	}
	originalID := user.ID()
	fmt.Printf("Original UserID: %s\n", originalID.String())

	t.Run("Text Marshaling", func(t *testing.T) {
		text, err := originalID.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText() error = %v", err)
		}
		if len(text) == 0 {
			t.Fatal("MarshalText() returned empty byte slice")
		}

		// Use UserIDFromString which uses UnmarshalText internally.
		parsedID, err := passkeys.UserIDFromString(string(text))
		if err != nil {
			t.Fatalf("UserIDFromString() error = %v", err)
		}

		if !reflect.DeepEqual(originalID, parsedID) {
			t.Errorf("text marshaling round-trip failed: got %v, want %v", parsedID, originalID)
		}
	})

	t.Run("Binary Marshaling", func(t *testing.T) {
		// Get binary representation from the user object.
		binaryData := user.WebAuthnID()

		// Use UserIDFromBytes which uses UnmarshalBinary internally.
		parsedID, err := passkeys.UserIDFromBytes(binaryData)
		if err != nil {
			t.Fatalf("UserIDFromBytes() error = %v", err)
		}

		if !reflect.DeepEqual(originalID, parsedID) {
			t.Errorf("binary marshaling round-trip failed: got %v, want %v", parsedID, originalID)
		}
	})

	t.Run("String Conversion", func(t *testing.T) {
		idStr := originalID.String()
		if len(idStr) == 0 {
			t.Fatal("String() returned empty string")
		}

		parsedID, err := passkeys.UserIDFromString(idStr)
		if err != nil {
			t.Fatalf("UserIDFromString() error = %v", err)
		}

		if !reflect.DeepEqual(originalID, parsedID) {
			t.Errorf("string conversion round-trip failed: got %v, want %v", parsedID, originalID)
		}
	})

	t.Run("Error Cases", func(t *testing.T) {
		_, err := passkeys.UserIDFromString("not-valid-base64-!")
		if err == nil {
			t.Error("expected error for invalid base64 string, got nil")
		}

		_, err = passkeys.UserIDFromString("short")
		if err == nil {
			t.Error("expected error for short string, got nil")
		}

		_, err = passkeys.UserIDFromBytes([]byte{1, 2, 3})
		if err == nil {
			t.Error("expected error for short byte slice, got nil")
		}
	})
}

func TestUser_CredentialManagement(t *testing.T) {
	user, err := passkeys.NewUser("test@example.com", "Test User")
	if err != nil {
		t.Fatal(err)
	}

	cred1 := webauthn.Credential{
		ID:        []byte("credential-id-1"),
		PublicKey: []byte("public-key-1"),
	}
	cred2 := webauthn.Credential{
		ID:        []byte("credential-id-2"),
		PublicKey: []byte("public-key-2"),
	}

	// Add a credential.
	user.AddCredential(cred1)
	if creds := user.WebAuthnCredentials(); len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if !bytes.Equal(user.WebAuthnCredentials()[0].ID, cred1.ID) {
		t.Errorf("added credential has wrong ID")
	}

	// Add a second credential.
	user.AddCredential(cred2)
	if creds := user.WebAuthnCredentials(); len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}

	// Update the first credential.
	updatedCred1 := webauthn.Credential{
		ID:        []byte("credential-id-1"),
		PublicKey: []byte("new-public-key-1"),
	}
	wasUpdated := user.UpdateCredential(updatedCred1)
	if !wasUpdated {
		t.Error("UpdateCredential() returned false, want true")
	}

	creds := user.WebAuthnCredentials()
	if !bytes.Equal(creds[0].PublicKey, updatedCred1.PublicKey) {
		t.Errorf("credential was not updated correctly")
	}
	// Ensure the other credential was not affected.
	if !bytes.Equal(creds[1].PublicKey, cred2.PublicKey) {
		t.Errorf("updating one credential affected another")
	}

	// Try to update a non-existent credential.
	nonExistentCred := webauthn.Credential{ID: []byte("non-existent-id")}
	wasUpdated = user.UpdateCredential(nonExistentCred)
	if wasUpdated {
		t.Error("UpdateCredential() returned true for non-existent credential, want false")
	}
}

func TestUser_ParseUID(t *testing.T) {
	user, err := passkeys.NewUser("test@example.com", "Test User")
	if err != nil {
		t.Fatal(err)
	}
	originalID := user.ID()
	idStr := originalID.String()

	parsedID, err := user.ParseUID(idStr)
	if err != nil {
		t.Fatalf("ParseUID() error = %v", err)
	}

	if !reflect.DeepEqual(originalID, parsedID) {
		t.Errorf("ParseUID() round-trip failed: got %v, want %v", parsedID, originalID)
	}

	_, err = user.ParseUID("invalid-string")
	if err == nil {
		t.Error("expected error when parsing invalid UID string, got nil")
	}
}
