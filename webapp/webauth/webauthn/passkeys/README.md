# Package [cloudeng.io/webapp/webauth/webauthn/passkeys](https://pkg.go.dev/cloudeng.io/webapp/webauth/webauthn/passkeys?tab=doc)

```go
import cloudeng.io/webapp/webauth/webauthn/passkeys
```

Package passkeys provides support for creating and authenticating WebAuthn
passkeys.

## Constants
### AuthenticationCookie, RegistrationCookie
```go
// AuthenticationCookie is set during the login/authentication
// webauthn flow (set in Begin and cleared in Finish).
AuthenticationCookie = cookies.Secure("webauthn_authentication")
// RegistrationCookie is set during the registration webauthn flow
// (set in Begin and cleared in Finish).
RegistrationCookie = cookies.Secure("webauthn_registration")

```



## Variables
### BeginDiscoverableAuthenticationEndpoint
```go
BeginDiscoverableAuthenticationEndpoint = jsonapi.Endpoint[struct{}, *protocol.CredentialAssertion]{}

```
BeginDiscoverableAuthenticationEndpoint represents the endpoint for
beginning the authentication using a discoverable passkey. The user's
identity will be determined by the user handle provided in the request.
The response will contain the options for the authentication request.

### BeginRegistrationEndpoint
```go
BeginRegistrationEndpoint = jsonapi.Endpoint[
	BeginRegistrationRequest,
	*protocol.PublicKeyCredentialCreationOptions]{}

```
BeginRegistrationEndpoint represents the endpoint for beginning the
registration process.

### FinishAuthenticationEndpoint
```go
FinishAuthenticationEndpoint = jsonapi.Endpoint[struct{}, struct{}]{}

```
FinishAuthenticationEndpoint represents the endpoint for finishing the
authentication process. It expects a request with a JSON body containing
the verification data as expected by the webauthn.FinishDiscoverableLogin
method, this method parses the request directly and hence the ParseRequest
is not used. The response on success is simply a http.StatusOK with an empty
body. Strictly speaking this variable is not used but serves to document the
endpoint.

### FinishRegistrationEndpoint
```go
FinishRegistrationEndpoint = jsonapi.Endpoint[struct{}, struct{}]{}

```
FinishRegistrationEndpoint represents the endpoint for finishing the
registration process. It expects a request with a JSON body containing the
verification data as expected by the webauthn.FinishRegistration method,
this method parses the request directly and hence the ParseRequest is not
used. The response on success is simply a http.StatusOK with an empty body.
Strictly speaking this variable is not used but serves to document the
endpoint.

### VerifyAuthenticationEndpoint
```go
VerifyAuthenticationEndpoint = jsonapi.Endpoint[struct{}, struct{}]{}

```
VerifyAuthenticationEndpoint represents the endpoint for verifying the
authentication of a user. It expects the user to be authenticated and to
have an entry in the user database.



## Types
### Type BeginRegistrationRequest
```go
type BeginRegistrationRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}
```
BeginRegistrationRequest represents the request body for beginning the
registration process, the client should send a JSON object with the user's
email address and display name.


### Type EmailValidator
```go
type EmailValidator interface {
	Validate(email string) error
}
```
EmailValidator defines an interface for validating email addresses.


### Type Handler
```go
type Handler struct {
	// contains filtered or unexported fields
}
```
Handler provides http Handlers that implement passkey registration and
authentication using the WebAuthn protocol. These endpoints accept JSON
requests and responses.

### Functions

```go
func NewHandler(w WebAuthn, sm SessionManager, um UserDatabase, lm LoginManager, opts ...HandlerOption) *Handler
```
NewHandler creates a new passkeys handler with the provided WebAuthn
implementation, session and user managers.



### Methods

```go
func (h *Handler) BeginDiscoverableAuthentication(rw http.ResponseWriter, _ *http.Request)
```


```go
func (h *Handler) BeginRegistration(rw http.ResponseWriter, r *http.Request)
```
BeginRegistration starts the registration process for a user. It expects a
request with a JSON body containing the user's email address.


```go
func (h *Handler) FinishAuthentication(rw http.ResponseWriter, r *http.Request)
```


```go
func (h *Handler) FinishRegistration(rw http.ResponseWriter, r *http.Request)
```


```go
func (h *Handler) VerifyAuthentication(rw http.ResponseWriter, r *http.Request)
```




### Type HandlerOption
```go
type HandlerOption func(*options)
```
HandlerOption represents an option for configuring the Handler.

### Functions

```go
func WithEmailValidator(validator EmailValidator) HandlerOption
```
WithEmailValidator sets the email validator for the handler.


```go
func WithLogger(logger *slog.Logger) HandlerOption
```
WithLogger sets the logger for the handler. The default is discard log
output.


```go
func WithMediation(mediation protocol.CredentialMediationRequirement) HandlerOption
```
WithMediation sets the mediation requirement for the handler.


```go
func WithRegistrationOptions(opts ...webauthn.RegistrationOption) HandlerOption
```
WithRegistrationOptions sets the registration options for the handler.


```go
func WithSessionCookieScopeAndDuration(ck cookies.ScopeAndDuration) HandlerOption
```
WithSessionCookieScopeAndDuration sets the session cookie's scope (domain,
path) and duration.




### Type JWTCookieLoginManager
```go
type JWTCookieLoginManager struct {

	// LoginCookie is set when the user has successfully logged in using
	// webauthn and is used to inform the server that the user has
	// successfully logged in
	LoginCookie cookies.Secure // initialized as cookies.T("webauthn_login")
	// contains filtered or unexported fields
}
```
JWTCookieLoginManager implements the LoginManager interface using JWTs
stored in cookies.

### Functions

```go
func NewJWTCookieLoginManager(signer jwtutil.Signer, issuer string, cookie cookies.ScopeAndDuration) JWTCookieLoginManager
```
NewJWTCookieLoginManager creates a new JWTCookieLoginManager instance.



### Methods

```go
func (m JWTCookieLoginManager) AuthenticateUser(r *http.Request) (UserID, error)
```


```go
func (m JWTCookieLoginManager) UserAuthenticated(rw http.ResponseWriter, user UserID) error
```




### Type LoginManager
```go
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
```
LoginManager defines the interface for managing logged in users who have
authenticated using a passkey.


### Type RAMUserDatabase
```go
type RAMUserDatabase struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewRAMUserDatabase() *RAMUserDatabase
```



### Methods

```go
func (sm RAMUserDatabase) Authenticated(tmpKey string) (sessionData *webauthn.SessionData, err error)
```


```go
func (sm RAMUserDatabase) Authenticating(sessionData *webauthn.SessionData) (tmpKey string, err error)
```


```go
func (um RAMUserDatabase) Lookup(userID UserID) (*User, error)
```


```go
func (sm RAMUserDatabase) Registered(tmpKey string) (user *User, sessionData *webauthn.SessionData, err error)
```


```go
func (sm RAMUserDatabase) Registering(user *User, sessionData *webauthn.SessionData) (tmpKey string, exists bool, err error)
```


```go
func (um RAMUserDatabase) Store(user *User) error
```




### Type SessionManager
```go
type SessionManager interface {
	// Used when creating a new passkey.
	Registering(user *User, sessionData *webauthn.SessionData) (tmpKey string, exists bool, err error)
	Registered(tmpKey string) (user *User, sessionData *webauthn.SessionData, err error)

	// Used when authenticating a passkey.
	Authenticating(sessionData *webauthn.SessionData) (tmpKey string, err error)
	Authenticated(tmpKey string) (sessionData *webauthn.SessionData, err error)
}
```
SessionManager is the interface used by passkeys.Server to manage state
between 'begin' and 'finish' registration and authentication requests.


### Type User
```go
type User struct {
	// contains filtered or unexported fields
}
```
User represents a user that registers to use a passkey and implements
webauthn.User

### Functions

```go
func NewUser(email, displayName string) (*User, error)
```
NewUser creates a new user with the given email and display name.



### Methods

```go
func (u *User) AddCredential(cred webauthn.Credential)
```
Implements webauthn.User.


```go
func (u *User) ID() UserID
```
ID returns the unique identifier for the user.


```go
func (u User) ParseUID(uid string) (UserID, error)
```
ParseUID parses a string representation of a UserID and returns the UserID.
It returns an error if the string cannot be parsed. It is required to parse
a UserID into the implementation of UserID used by the User struct.


```go
func (u *User) UpdateCredential(cred webauthn.Credential) bool
```
UpdateCredential updates an existing credential for the user.


```go
func (u *User) WebAuthnCredentials() []webauthn.Credential
```
Implements webauthn.User.


```go
func (u *User) WebAuthnDisplayName() string
```
Implements webauthn.User.


```go
func (u *User) WebAuthnID() []byte
```
Implements webauthn.User.


```go
func (u *User) WebAuthnName() string
```
Implements webauthn.User.




### Type UserDatabase
```go
type UserDatabase interface {

	// Store persists the user in the database, using the user.ID().String() as the key.
	Store(user *User) error

	// Lookup retrieves a user using the UUID it was original created with.
	Lookup(uid UserID) (*User, error)
}
```
UserDatabase is an interface for a user database that supports registering
and authenticating passkeys.


### Type UserID
```go
type UserID interface {
	String() string               // Returns a string representation of the user ID that can be used usable as a key in a map. String should return the same value as MarshalText and hence UnmarshalText(String()) == UnmarshalText(MarshalText()).
	UnmarshalBinary([]byte) error // Converts a byte slice to a UserID.
	MarshalText() ([]byte, error) // Converts the UserID to base64.RawURLEncoding representation.
	UnmarshalText([]byte) error   // Converts a base64.RawURLEncoding text representation to a UserID.
}
```
UserID is used to uniquely identify users in the passkey system. It must
be a cryptographically secure randomly generated value, (eg. 64 bytes read
crypto.rand.Reader).

### Functions

```go
func UserIDFromBytes(b []byte) (UserID, error)
```
UserIDFromBytes creates a UserID from a byte slice.


```go
func UserIDFromString(s string) (UserID, error)
```
UserIDFromString creates a UserID from a base64.RawURLEncoding string.




### Type WebAuthn
```go
type WebAuthn interface {
	BeginMediatedRegistration(user webauthn.User, mediation protocol.CredentialMediationRequirement, opts ...webauthn.RegistrationOption) (creation *protocol.CredentialCreation, session *webauthn.SessionData, err error)
	FinishRegistration(user webauthn.User, session webauthn.SessionData, r *http.Request) (*webauthn.Credential, error)
	BeginDiscoverableMediatedLogin(mediation protocol.CredentialMediationRequirement, opts ...webauthn.LoginOption) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	FinishPasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, request *http.Request) (user webauthn.User, credential *webauthn.Credential, err error)
}
```
WebAuthn defines the subset of webauthn.WebAuthn used by this package.





