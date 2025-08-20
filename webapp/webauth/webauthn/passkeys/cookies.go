// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys

import (
	"cloudeng.io/webapp/cookies"
)

const (
	// AuthenticationCookie is set during the login/authentication
	// webauthn flow (set in Begin and cleared in Finish).
	AuthenticationCookie = cookies.Secure("webauthn_authentication")
	// RegistrationCookie is set during the registration webauthn flow
	// (set in Begin and cleared in Finish).
	RegistrationCookie = cookies.Secure("webauthn_registration")
)
