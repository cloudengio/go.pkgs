// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

// UserID is used to uniquely identify users in the passkey system.
// It must be a cryptographically secure randomly generated value,
// (eg. 64 bytes read crypto.rand.Reader).
type UserID interface {
	String() string               // Returns a string representation of the user ID that can be used usable as a key in a map. String should return the same value as MarshalText and hence UnmarshalText(String()) == UnmarshalText(MarshalText()).
	UnmarshalBinary([]byte) error // Converts a byte slice to a UserID.
	MarshalText() ([]byte, error) // Converts the UserID to base64.RawURLEncoding representation.
	UnmarshalText([]byte) error   // Converts a base64.RawURLEncoding text representation to a UserID.
}

type uuidUserID [64]byte

func (u uuidUserID) String() string {
	return base64.RawURLEncoding.EncodeToString(u[:])
}

func (u uuidUserID) MarshalText() ([]byte, error) {
	buf := make([]byte, base64.RawURLEncoding.EncodedLen(len(u[:])))
	base64.RawURLEncoding.Encode(buf, u[:])
	return buf, nil
}

func (u *uuidUserID) UnmarshalText(text []byte) error {
	n, err := base64.RawURLEncoding.Decode(u[:], text)
	if err != nil {
		return fmt.Errorf("failed to parse uid from text: %w", err)
	}
	if n != 64 {
		return fmt.Errorf("invalid length for uid: %d, expected 64", n)
	}
	return nil
}

func (u *uuidUserID) UnmarshalBinary(b []byte) error {
	if len(b) != 64 {
		return fmt.Errorf("invalid byte slice length for uid: %d, expected 64", len(b))
	}
	copy(u[:], b)
	return nil
}

// UserIDFromBytes creates a UserID from a byte slice.
func UserIDFromBytes(b []byte) (UserID, error) {
	var id uuidUserID
	if err := id.UnmarshalBinary(b); err != nil {
		return nil, fmt.Errorf("failed to create UserID from bytes: %w", err)
	}
	return &id, nil
}

// UserIDFromString creates a UserID from a base64.RawURLEncoding string.
func UserIDFromString(s string) (UserID, error) {
	var id uuidUserID
	if err := id.UnmarshalText([]byte(s)); err != nil {
		return nil, fmt.Errorf("failed to create UserID from string: %w", err)
	}
	return &id, nil
}

// User represents a user that registers to use a passkey and implements webauthn.User
type User struct {
	id          [64]byte              // Unique identifier for the user, used in WebAuthn.
	email       string                // Email address of the user, supplied by the user and not verfied in any way.
	displayName string                // Display name of the user, can be used for UI purposes.
	credentials []webauthn.Credential // List of WebAuthn credentials associated with the user.
}

// NewUser creates a new user with the given email and display name.
func NewUser(email, displayName string) (*User, error) {
	u := &User{
		email:       email,
		displayName: displayName,
	}
	n, err := rand.Read(u.id[:]) // Fill the id with random bytes.
	if err != nil {
		return nil, fmt.Errorf("failed to generate random user ID: %w", err)
	}
	if n != 64 {
		return nil, fmt.Errorf("failed to generate enough random data: got %v, want 64", n)
	}
	return u, nil
}

// ID returns the unique identifier for the user.
func (u *User) ID() UserID {
	tmp := uuidUserID(u.id)
	return &tmp
}

// UpdateCredential updates an existing credential for the user.
func (u *User) UpdateCredential(cred webauthn.Credential) bool {
	for i, c := range u.credentials {
		if bytes.Equal(c.ID, cred.ID) {
			u.credentials[i] = cred
			return true
		}
	}
	return false
}

// ParseUID parses a string representation of a UserID and returns the UserID.
// It returns an error if the string cannot be parsed.
// It is required to parse a UserID into the implementation of UserID
// used by the User struct.
func (u User) ParseUID(uid string) (UserID, error) {
	var id uuidUserID
	if err := id.UnmarshalText([]byte(uid)); err != nil {
		return nil, fmt.Errorf("failed to parse UserID from string: %w", err)
	}
	return &id, nil
}

// Implements webauthn.User.
func (u *User) WebAuthnID() []byte {
	return u.id[:]
}

// Implements webauthn.User.
func (u *User) WebAuthnName() string {
	return u.email
}

// Implements webauthn.User.
func (u *User) WebAuthnDisplayName() string {
	return u.displayName
}

// Implements webauthn.User.
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// Implements webauthn.User.
func (u *User) AddCredential(cred webauthn.Credential) {
	u.credentials = append(u.credentials, cred)
}
