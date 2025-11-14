// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys

import (
	"fmt"
	"sync"
)

// UserDatabase is an interface for a user database that supports registering
// and authenticating passkeys.
type UserDatabase interface {

	// Store persists the user in the database, using the user.ID().String() as the key.
	Store(user *User) error

	// Lookup retrieves a user using the UUID it was original created with.
	Lookup(uid UserID) (*User, error)
}

type RAMUserDatabase struct {
	*sessionManager
	*userDatabase
}

func NewRAMUserDatabase() *RAMUserDatabase {
	return &RAMUserDatabase{
		sessionManager: &sessionManager{
			sessions: make(map[string]sessionState),
		},
		userDatabase: &userDatabase{
			users: make(map[string]User),
		},
	}
}

type userDatabase struct {
	mu    sync.Mutex
	users map[string]User // Maps user IDs to user data.
}

func (um *userDatabase) Store(user *User) error {
	um.mu.Lock()
	defer um.mu.Unlock()
	um.users[user.ID().String()] = *user
	return nil
}

func (um *userDatabase) Lookup(userID UserID) (*User, error) {
	um.mu.Lock()
	defer um.mu.Unlock()
	user, exists := um.users[userID.String()]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}
