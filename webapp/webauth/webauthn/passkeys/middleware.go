// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys

import (
	"errors"
	"net/http"
	"time"

	"cloudeng.io/webapp/cookies"
	"cloudeng.io/webapp/webauth/jwtutil"
	"github.com/golang-jwt/jwt/v5"
)

// LoginManager defines the interface for managing logged in users who
// have authenticated using a passkey.
type LoginManager interface {
	// UserAuthenticated is called after a user has successfully logged in with a passkey.
	// It should be used to set a session Cookie, or a JWT token to be validated
	// on subsequent requests. The expiration parameter indicates how long the
	// login session should be valid.
	UserAuthenticated(rw http.ResponseWriter, user UserID) error

	// AuthenticateUser is called to validate the user based on the request.
	// It should return the UserID of the authenticated user or an error if authentication fails.
	AuthenticateUser(r *http.Request) (UserID, error)
}

// JWTCookieLoginManager implements the LoginManager interface using JWTs stored in cookies.
type JWTCookieLoginManager struct {
	signer jwtutil.Signer
	claims jwt.RegisteredClaims
	parser *jwt.Parser

	loginCookie cookies.ScopeAndDuration
	// LoginCookie is set when the user has successfully logged in using
	// webauthn and is used to inform the server that the user has
	// successfully logged in
	LoginCookie cookies.Secure // initialized as cookies.T("webauthn_login")
}

// NewJWTCookieLoginManager creates a new JWTCookieLoginManager instance.
func NewJWTCookieLoginManager(signer jwtutil.Signer, issuer string, cookie cookies.ScopeAndDuration) JWTCookieLoginManager {
	p := jwt.NewParser(
		jwt.WithIssuer(issuer),
		jwt.WithAudience("webauthn"),
	)
	m := JWTCookieLoginManager{
		signer:      signer,
		loginCookie: cookie.SetDefaults("", "/", 10*time.Minute),
		claims: jwt.RegisteredClaims{
			Issuer:   issuer,
			Audience: jwt.ClaimStrings{"webauthn"},
		},
		parser:      p,
		LoginCookie: cookies.Secure("webauthn_login"),
	}
	return m
}

func (m JWTCookieLoginManager) UserAuthenticated(rw http.ResponseWriter, user UserID) error {
	now := time.Now()

	// Create the JWT claims.
	claims := m.claims
	claims.Subject = user.String()
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(m.loginCookie.Duration))
	claims.NotBefore = jwt.NewNumericDate(now)

	tokenString, err := m.signer.Sign(claims)
	if err != nil {
		return err
	}
	m.LoginCookie.Set(rw, m.loginCookie.Cookie(tokenString))
	return nil
}

func (m JWTCookieLoginManager) AuthenticateUser(r *http.Request) (UserID, error) {
	tokenString, ok := m.LoginCookie.Read(r)
	if !ok {
		return nil, errors.New("missing authentication cookie")
	}

	var claims jwt.RegisteredClaims
	_, err := m.parser.ParseWithClaims(tokenString, &claims, m.signer.KeyFunc)
	if err != nil {
		return nil, err
	}
	uid, err := UserIDFromString(claims.Subject)
	if err != nil {
		return nil, err
	}

	return uid, nil
}
