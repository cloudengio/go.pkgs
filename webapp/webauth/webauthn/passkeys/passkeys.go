// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package passkeys provides support for creating and authenticating WebAuthn passkeys.
package passkeys

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"cloudeng.io/webapp/jsonapi"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthn defines the subset of webauthn.WebAuthn used by this package.
type WebAuthn interface {
	BeginMediatedRegistration(user webauthn.User, mediation protocol.CredentialMediationRequirement, opts ...webauthn.RegistrationOption) (creation *protocol.CredentialCreation, session *webauthn.SessionData, err error)
	FinishRegistration(user webauthn.User, session webauthn.SessionData, r *http.Request) (*webauthn.Credential, error)
	BeginDiscoverableMediatedLogin(mediation protocol.CredentialMediationRequirement, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	FinishPasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, response *http.Request) (user webauthn.User, credential *webauthn.Credential, err error)
}

// EmailValidator defines an interface for validating email addresses.
type EmailValidator interface {
	Validate(email string) error
}

// Handler provides http Handlers that implement passkey registration and authentication
// using the WebAuthn protocol. These endpoints accept JSON requests and responses.
type Handler struct {
	w    WebAuthn
	sm   SessionManager
	um   UserDatabase
	mw   Middleware
	opts options
}

type options struct {
	logger                *slog.Logger
	sessionCookieDuration time.Duration
	loginCookieDuration   time.Duration
	emailValidator        EmailValidator
}

// HandlerOption represents an option for configuring the Handler.
type HandlerOption func(*options)

// WithLogger sets the logger for the handler.
// The default is discard log output.
func WithLogger(logger *slog.Logger) HandlerOption {
	return func(o *options) {
		o.logger = logger
	}
}

// WithSessionCookieDuration sets the duration for session cookies,
// that is, those used during being/finish endpoint invocations.
// The default is 10 minutes.
func WithSessionCookieDuration(duration time.Duration) HandlerOption {
	return func(o *options) {
		o.sessionCookieDuration = duration
	}
}

// WithLoginCookieDuration sets the duration for login cookies.
// The default is 7 days.
func WithLoginCookieDuration(duration time.Duration) HandlerOption {
	return func(o *options) {
		o.loginCookieDuration = duration
	}
}

func WithEmailValidator(validator EmailValidator) HandlerOption {
	return func(o *options) {
		o.emailValidator = validator
	}
}

// NewHandler creates a new passkeys handler with the provided WebAuthn
// implementation, session and user managers.
func NewHandler(w WebAuthn, sm SessionManager, um UserDatabase, mw Middleware, opts ...HandlerOption) *Handler {
	h := &Handler{
		w:  w,
		um: um,
		sm: sm,
		mw: mw,
	}
	for _, fn := range opts {
		fn(&h.opts)
	}
	if h.opts.sessionCookieDuration == 0 {
		h.opts.sessionCookieDuration = 10 * time.Minute // Default session cookie duration.
	}
	if h.opts.loginCookieDuration == 0 {
		h.opts.loginCookieDuration = 7 * 24 * time.Hour // Default login cookie duration.
	}
	if h.opts.logger == nil {
		h.opts.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return h
}

// BeginRegistrationEndpoint represents the endpoint for beginning the registration process.
var BeginRegistrationEndpoint = jsonapi.Endpoint[
	BeginRegistrationRequest,
	*protocol.PublicKeyCredentialCreationOptions]{}

// BeginRegistrationRequest represents the request body for beginning the registration process,
// the client should send a JSON object with the user's email address and display name.
type BeginRegistrationRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// BeginRegistration starts the registration process for a user.
// It expects a request with a JSON body containing the user's email address.
func (h *Handler) BeginRegistration(rw http.ResponseWriter, r *http.Request,
	mediationRequirement protocol.CredentialMediationRequirement, opts ...webauthn.RegistrationOption) {
	var req BeginRegistrationRequest
	logger := h.opts.logger.With("method", "BeginRegistration")

	if err := BeginRegistrationEndpoint.ParseRequest(rw, r, &req); err != nil {
		logger.Error("failed to parse request", "err", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to parse request", http.StatusBadRequest)
		return
	}

	if h.opts.emailValidator != nil {
		if err := h.opts.emailValidator.Validate(req.Email); err != nil {
			logger.Error("invalid email address", "email", req.Email, "err", err.Error())
			jsonapi.WriteErrorMsg(rw, "invalid email address", http.StatusBadRequest)
			return
		}
	}

	user, err := NewUser(req.Email, req.DisplayName)
	if err != nil {
		logger.Error("failed to create user", "err", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to create user", http.StatusInternalServerError)
		return
	}

	creds, session, err := h.w.BeginMediatedRegistration(
		user, mediationRequirement, opts...)
	if err != nil {
		logger.Error("failed to begin mediated registration", "err", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to begin mediated registration", http.StatusInternalServerError)
		return
	}

	tmpKey, exists, err := h.sm.Registering(user, session)
	if err != nil {
		logger.Error("failed to begin registration", "err", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to begin registration", http.StatusInternalServerError)
		return
	}
	if exists {
		logger.Error("user already registered", "webauthn_name", user.WebAuthnName())
		jsonapi.WriteErrorMsg(rw, "user already registered", http.StatusConflict)
		return
	}

	RegistrationCookie.SetSecureWithExpiration(rw, tmpKey, h.opts.sessionCookieDuration)
	if err := BeginRegistrationEndpoint.WriteResponse(rw, &creds.Response); err != nil {
		logger.Error("failed to write response", "error", err.Error())
		return
	}
	logger.Info("registration started", "user_id", user.ID().String(), "webauthn_name", user.WebAuthnName())
}

// FinishRegistrationEndpoint represents the endpoint for finishing the registration process.
// It expects a request with a JSON body containing the verification data as expected
// by the webauthn.FinishRegistration method, this method parses the request directly
// and hence the ParseRequest is not used. The response on success is simply
// a http.StatusOK with an empty body. Strictly speaking this variable is not used
// but serves to document the endpoint.
var FinishRegistrationEndpoint = jsonapi.Endpoint[struct{}, struct{}]{}

func webauthError(err error) slog.Attr {
	if werr, ok := err.(*protocol.Error); ok {
		return slog.Group("webauthn_error", "type", werr.Type, "details", werr.Details, "dev_info", werr.DevInfo, "err", werr.Err)
	}
	return slog.Attr{}
}

func (h *Handler) FinishRegistration(rw http.ResponseWriter, r *http.Request) {
	logger := h.opts.logger.With("method", "FinishRegistration")

	sessionKey, ok := RegistrationCookie.ReadAndClear(rw, r)
	if !ok {
		logger.Error("missing registration cookie")
		jsonapi.WriteErrorMsg(rw, "missing registration cookie", http.StatusBadRequest)
		return
	}

	user, sessionData, err := h.sm.Registered(sessionKey)
	if err != nil {
		logger.Error("failed to retrieve session data", "session_key", sessionKey, "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to retrieve session data", http.StatusInternalServerError)
		return
	}

	if user == nil || sessionData == nil {
		logger.Error("invalid session data", "session_key", sessionKey)
		jsonapi.WriteErrorMsg(rw, "invalid session data", http.StatusBadRequest)
		return
	}
	cred, err := h.w.FinishRegistration(user, *sessionData, r)
	if err != nil {
		logger.Error("failed to finish registration", "error", err.Error(), webauthError(err))
		jsonapi.WriteErrorMsg(rw, "failed to finish registration", http.StatusBadRequest)
		return
	}

	user.AddCredential(*cred)

	if err := h.um.Store(user); err != nil {
		logger.Error("failed to store user", "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to store user", http.StatusInternalServerError)
		return
	}

	logger.Info("registration successful", "user_id", user.ID(), "webauthn_name", user.WebAuthnName())
}

// BeginDiscoverableAuthenticationEndpoint represents the endpoint for beginning
// the authentication using a discoverable passkey. The user's identity will be
// determined by the user handle provided in the request. The response will contain
// the options for the authentication request.
var BeginDiscoverableAuthenticationEndpoint = jsonapi.Endpoint[struct{}, *protocol.PublicKeyCredentialRequestOptions]{}

func (h *Handler) BeginDiscoverableAuthentication(rw http.ResponseWriter, r *http.Request) {
	logger := h.opts.logger.With("method", "BeginDiscoverableAuthentication")

	options, session, err := h.w.BeginDiscoverableMediatedLogin(protocol.MediationDefault)
	if err != nil {
		logger.Error("failed to begin discoverable authentication", "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to begin discoverable authentication", http.StatusInternalServerError)
		return
	}
	tmpKey, err := h.sm.Authenticating(session)
	if err != nil {
		logger.Error("failed to begin authenticating", "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to begin authenticating", http.StatusInternalServerError)
		return
	}
	AuthenticationCookie.SetSecureWithExpiration(rw, tmpKey, h.opts.sessionCookieDuration)

	if err := BeginDiscoverableAuthenticationEndpoint.WriteResponse(rw, &options.Response); err != nil {
		logger.Error("failed to write response", "error", err.Error())
		return
	}
	logger.Info("discoverable authentication started", "tmp_key", tmpKey)
}

// FinishAuthenticationEndpoint represents the endpoint for finishing the authentication process.
// It expects a request with a JSON body containing the verification data as expected
// by the webauthn.FinishDiscoverableLogin method, this method parses the request directly
// and hence the ParseRequest is not used. The response on success is simply
// a http.StatusOK with an empty body. Strictly speaking this variable is not used
// but serves to document the endpoint.
var FinishAuthenticationEndpoint = jsonapi.Endpoint[struct{}, struct{}]{}

func (h *Handler) FinishAuthentication(rw http.ResponseWriter, r *http.Request) {
	logger := h.opts.logger.With("method", "FinishAuthentication")

	sessionKey, ok := AuthenticationCookie.ReadAndClear(rw, r)
	if !ok {
		logger.Error("missing authentication cookie")
		jsonapi.WriteErrorMsg(rw, "missing authentication cookie", http.StatusBadRequest)
		return
	}

	session, err := h.sm.Authenticated(sessionKey)
	if err != nil {
		logger.Error("failed to retrieve session data", "session_key", sessionKey, "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to retrieve session data", http.StatusInternalServerError)
		return
	}

	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		uid, err := UserIDFromBytes(userHandle)
		if err != nil {
			logger.Error("invalid user handle", "user_handle", userHandle, "error", err.Error())
			return nil, fmt.Errorf("invalid user handle: %w", err)
		}
		user, err := h.um.Lookup(uid)
		if err != nil {
			logger.Error("failed to lookup user", "user_id", uid.String(), "error", err.Error())
			return nil, fmt.Errorf("failed to lookup user: %w", err)
		}
		return user, nil
	}
	user, cred, err := h.w.FinishPasskeyLogin(handler, *session, r)
	if err != nil {
		logger.Error("failed to finish discoverable login", "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to finish discoverable login", http.StatusInternalServerError)
		return
	}

	pu, ok := user.(*User)
	if !ok {
		logger.Error("user is not of type *User", "user_type", fmt.Sprintf("%T", user))
		jsonapi.WriteErrorMsg(rw, "internal server error", http.StatusInternalServerError)
		return
	}
	if !pu.UpdateCredential(*cred) {
		logger.Error("failed to update credential for user", "user_id", pu.ID().String(), "credential_id", base64.RawURLEncoding.EncodeToString(cred.ID))
		jsonapi.WriteErrorMsg(rw, "failed to update credential", http.StatusInternalServerError)
		return
	}

	if err := h.um.Store(pu); err != nil {
		logger.Error("failed to store user", "user_id", pu.ID().String(), "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to store user", http.StatusInternalServerError)
		return
	}

	if err := h.mw.UserAuthenticated(rw, pu.ID(), h.opts.loginCookieDuration); err != nil {
		logger.Error("failed to set authenticated session", "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to set authenticated session", http.StatusInternalServerError)
		return
	}

}

// VerifyAuthenticationEndpoint represents the endpoint for verifying the authentication
// of a user. It expects the user to be authenticated and to have an entry in the user database.
var VerifyAuthenticationEndpoint = jsonapi.Endpoint[struct{}, struct{}]{}

func (h *Handler) VerifyAuthentication(rw http.ResponseWriter, r *http.Request) {
	logger := h.opts.logger.With("method", "VerifyAuthentication")
	uid, err := h.mw.AuthenticateUser(r)
	if err != nil {
		logger.Error("failed to authenticate user", "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to authenticate user", http.StatusUnauthorized)
		return
	}
	user, err := h.um.Lookup(uid)
	if err != nil {
		logger.Error("failed to lookup user", "user_id", uid.String(), "error", err.Error())
		jsonapi.WriteErrorMsg(rw, "failed to lookup user", http.StatusInternalServerError)
		return
	}
	logger.Info("user authenticated", "user_id", uid.String(), "webauthn_name", user.WebAuthnName())
}
