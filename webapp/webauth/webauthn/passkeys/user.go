// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys

import (
	"bytes"
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// UserID is used to uniquely identify users in the passkey system.
// It must be a cryptographically secure randomly generated value, (eg a UUID,
// or the output crypto.rand.Reader).
type UserID interface {
	String() string               // Returns a string representation of the user ID that can be used usable as a key in a map. String should return the same value as MarshalText and hence UnmarshalText(String()) == UnmarshalText(MarshalText()).
	UnmarshalBinary([]byte) error // Converts a byte slice to a UserID.
	MarshalText() ([]byte, error) // Converts the UserID to a text representation.
	UnmarshalText([]byte) error   // Converts a text representation to a UserID.
}

type uuidUserID uuid.UUID

func (u uuidUserID) String() string {
	return uuid.UUID(u).String()
}

func (u uuidUserID) MarshalText() ([]byte, error) {
	return []byte(uuid.UUID(u).String()), nil
}

func (u *uuidUserID) UnmarshalText(text []byte) error {
	id, err := uuid.Parse(string(text))
	if err != nil {
		return fmt.Errorf("failed to parse UUID from text: %w", err)
	}
	*u = uuidUserID(id)
	return nil
}

func (u *uuidUserID) UnmarshalBinary(b []byte) error {
	if len(b) != 16 {
		return fmt.Errorf("invalid byte slice length for UUID: %d", len(b))
	}
	id, err := uuid.FromBytes(b)
	if err != nil {
		return fmt.Errorf("failed to convert bytes to UUID: %w", err)
	}
	*u = uuidUserID(id)
	return nil
}

func UserIDFromBytes(b []byte) (UserID, error) {
	var id uuidUserID
	if err := id.UnmarshalBinary(b); err != nil {
		return nil, fmt.Errorf("failed to create UserID from bytes: %w", err)
	}
	return &id, nil
}

// User represents a user that registers to use a passkey and implements webauthn.User
type User struct {
	id          uuid.UUID             // Unique identifier for the user, used in WebAuthn.
	email       string                // Email address of the user, supplied by the user and not verfied in any way.
	displayName string                // Display name of the user, can be used for UI purposes.
	credentials []webauthn.Credential // List of WebAuthn credentials associated with the user.
}

func NewUser(email, displayName string) (*User, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
	return &User{
		id:          id,
		email:       email,
		displayName: displayName,
	}, nil
}

func (u *User) ID() UserID {
	tmp := uuidUserID(u.id)
	return &tmp
}

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
