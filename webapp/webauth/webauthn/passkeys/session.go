// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys

import (
	"fmt"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

// SessionManager is the interface used by passkeys.Server to manage state
// between 'begin' and 'finish' registration and authentication requests.
type SessionManager interface {
	// Used when creating a new passkey.
	Registering(user *User, sessionData *webauthn.SessionData) (tmpKey string, exists bool, err error)
	Registered(tmpKey string) (user *User, sessionData *webauthn.SessionData, err error)

	// Used when authenticating a passkey.
	Authenticating(sessionData *webauthn.SessionData) (tmpKey string, err error)
	Authenticated(tmpKey string) (sessionData *webauthn.SessionData, err error)
}

type sessionState struct {
	user    *User
	session *webauthn.SessionData
}

type sessionManager struct {
	mu       sync.Mutex
	sessions map[string]sessionState // Maps temporary keys to state.
}

func (sm *sessionManager) Registering(user *User, sessionData *webauthn.SessionData) (tmpKey string, exists bool, err error) {
	tmpKey = uuid.New().String()
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[tmpKey] = sessionState{user: user, session: sessionData}
	return tmpKey, false, nil
}

func (sm *sessionManager) Registered(tmpKey string) (user *User, sessionData *webauthn.SessionData, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	data, exists := sm.sessions[tmpKey]
	if !exists {
		return nil, nil, fmt.Errorf("user not found")
	}
	delete(sm.sessions, tmpKey) // Remove the session after retrieval.
	return data.user, data.session, nil
}

func (sm *sessionManager) Authenticating(sessionData *webauthn.SessionData) (tmpKey string, err error) {
	tmpKey = uuid.New().String()
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[tmpKey] = sessionState{session: sessionData}
	return tmpKey, nil
}

func (sm *sessionManager) Authenticated(tmpKey string) (sessionData *webauthn.SessionData, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	data, exists := sm.sessions[tmpKey]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	delete(sm.sessions, tmpKey) // Remove the session after retrieval.
	return data.session, nil
}
