// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:generate tsc --target es2017 testdata/passkeys.ts
package passkeys_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudeng.io/webapp"
	"cloudeng.io/webapp/cookies"
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/devtest/chromedputil"
	"cloudeng.io/webapp/webauth/jwtutil"
	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	browserWebauthn "github.com/chromedp/cdproto/webauthn"
	"github.com/chromedp/chromedp"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var serverURL *url.URL

func init() {
	var err error
	serverURL, err = url.Parse("https://localhost:8088")
	if err != nil {
		panic(fmt.Sprintf("Failed to parse server URL: %v", err))
	}
}

func runServer(ctx context.Context, t *testing.T, tmpDir string, w *passkeys.Handler, errCh chan error) error {
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := devtest.NewSelfSignedCertUsingMkcert(certFile, keyFile, "localhost"); err != nil {
		return fmt.Errorf("failed to create self-signed certificates: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./testdata")))
	mux.HandleFunc("/generate-registration-options", w.BeginRegistration)
	mux.HandleFunc("/verify-registration", w.FinishRegistration)
	mux.HandleFunc("/generate-authentication-options", w.BeginDiscoverableAuthentication)
	mux.HandleFunc("/verify-authentication", w.FinishAuthentication)
	mux.HandleFunc("/verify", w.VerifyAuthentication)
	mux.HandleFunc("/generate",
		devtest.NewJSServer("generate", nil, "passkeys.js").ServeJS)
	mux.HandleFunc("/login",
		devtest.NewJSServer("login", nil, "passkeys.js").ServeJS)

	cfg, err := webapp.TLSConfigUsingCertFiles(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %v", err)
	}

	t.Logf("Starting TLS server at %s\n", serverURL.Host)
	ln, srv, err := webapp.NewTLSServer(ctx, serverURL.Host, mux, cfg)
	if err != nil {
		return fmt.Errorf("failed to create TLS server: %v", err)
	}

	go func() {
		errCh <- webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
	}()

	return nil
}

func TestPasskeysServer(t *testing.T) {
	ctx, serverCancel := context.WithCancel(t.Context())
	defer serverCancel()

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Test Passkeys",
		RPID:          "localhost",
		RPOrigins:     []string{serverURL.String()},
	})
	if err != nil {
		t.Fatalf("Failed to create WebAuthn instance: %v", err)
	}
	var logged strings.Builder
	logger := slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, &logged), nil))
	db := passkeys.NewRAMUserDatabase()
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	signer := jwtutil.NewED25519Signer(pubKey, privKey, "pkid")
	scopeAndDuration := cookies.ScopeAndDuration{
		Domain:   "localhost",
		Path:     "/",
		Duration: 10 * time.Minute,
	}
	mw := passkeys.NewJWTCookieLoginManager(signer, "localhost", scopeAndDuration)
	requireResidentKey := true
	w := passkeys.NewHandler(wa, db, db, mw,
		passkeys.WithLogger(logger),
		passkeys.WithRegistrationOptions(
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				AuthenticatorAttachment: protocol.Platform,
				RequireResidentKey:      &requireResidentKey,
				ResidentKey:             protocol.ResidentKeyRequirementRequired,
				UserVerification:        protocol.VerificationPreferred,
			}),
		),
	)
	errCh := make(chan error, 1)
	if err := runServer(ctx, t, t.TempDir(), w, errCh); err != nil {
		t.Fatalf("Failed to run server: %v", err)
	}

	// Give the server a moment to start.
	time.Sleep(100 * time.Millisecond)

	ctx, cancel, authenticatorID := setupBrowser(t)
	defer cancel()
	defer chromedp.Run(ctx, browserWebauthn.RemoveVirtualAuthenticator(authenticatorID)) //nolint:errcheck

	listenCh := chromedputil.RunLoggingListener(ctx, logger,
		chromedputil.WithNetworkLogging(),
		chromedputil.WithConsoleLogging(),
		chromedputil.WithExceptionLogging(),
		chromedputil.WithAnyEventLogging(),
	)

	// Run tests for registration and login.
	regResult := testPasskeyRegistration(ctx, t)

	uid, err := passkeys.UserIDFromString(regResult.UserHandle)
	if err != nil {
		t.Fatalf("Failed to get user ID from string: %v", err)
	}

	user, err := db.Lookup(uid)
	if err != nil {
		t.Fatalf("Failed to lookup user: %v", err)
	}
	if user.WebAuthnName() != regResult.Email {
		t.Fatalf("User email does not match: got %v, want %v", user.WebAuthnName(), regResult.Email)
	}

	loginRes := testPasskeyLogin(ctx, t)

	if !loginRes.Success {
		time.Sleep(100 * time.Millisecond) // Allow time for events to propagate.
		t.Errorf("Login failed: %s", loginRes.Error)
	}

	if got, want := regResult.UserHandle, loginRes.UserHandle; got != want {
		t.Errorf("User handle does not match: got %v, want %v", got, want)
	}
	if got, want := regResult.PublicKeyID, loginRes.PublicKeyID; got != want {
		t.Errorf("Public key ID does not match: got %v, want %v", got, want)
	}

	if err := chromedp.Run(ctx, browserWebauthn.RemoveVirtualAuthenticator(authenticatorID)); err != nil {
		t.Errorf("Failed to remove virtual authenticator: %v", err)
	}
	cancel()       // Stop the browser context.
	serverCancel() // Stop the web server.
	if err := <-errCh; err != nil {
		// http.ErrServerClosed is the expected error on graceful shutdown.
		if err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}
	<-listenCh
}

func setupBrowser(t *testing.T) (context.Context, context.CancelFunc, browserWebauthn.AuthenticatorID) {
	t.Helper()
	ctx, cancel := chromedputil.WithContextForCI(context.Background(), chromedputil.AllocatorLoggingWithLevel(2), chromedp.WithLogf(t.Logf))

	authOptions := &browserWebauthn.VirtualAuthenticatorOptions{
		Protocol:                    browserWebauthn.AuthenticatorProtocolCtap2,
		Transport:                   browserWebauthn.AuthenticatorTransportInternal,
		HasResidentKey:              true,
		HasUserVerification:         true,
		IsUserVerified:              true,
		AutomaticPresenceSimulation: true,
	}

	var authenticatorID browserWebauthn.AuthenticatorID
	if err := chromedp.Run(ctx,
		browserWebauthn.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			authenticatorID, err = browserWebauthn.AddVirtualAuthenticator(authOptions).Do(ctx)
			return err
		}),
	); err != nil {
		cancel()
		t.Fatalf("Failed to set up virtual authenticator: %v", err)
	}

	err := chromedp.Run(ctx, chromedp.Navigate(serverURL.String()))
	if err != nil {
		cancel()
		t.Fatalf("Failed to navigate to server URL: %v, %v", serverURL.String(), err)
	}

	if err := chromedputil.SourceScript(ctx, serverURL.String()+"/passkeys.js"); err != nil {
		cancel()
		t.Fatalf("Failed to source passkeys.js: %v", err)
	}

	return ctx, cancel, authenticatorID
}

type registrationResult struct {
	UserHandle  string `json:"user_handle"`
	PublicKeyID string `json:"public_key_id"`
	Email       string `json:"email"`
	Error       string `json:"error"`
}

func testPasskeyRegistration(ctx context.Context, t *testing.T) registrationResult {
	var result registrationResult
	err := chromedp.Run(ctx,
		// Call the registration function from the script.
		chromedp.Evaluate(`createPasskey('test@example.com', 'Test User').then((result) => { return result; });`, &result, chromedputil.WaitForPromise),
	)
	if err != nil {
		t.Fatalf("Passkey registration test failed: %v", err)
	}
	return result
}

type loginResult struct {
	Success     bool   `json:"success"`
	UserHandle  string `json:"user_handle"`
	PublicKeyID string `json:"public_key_id"`
	Error       string `json:"error"`
}

func testPasskeyLogin(ctx context.Context, t *testing.T) loginResult {
	var result loginResult
	err := chromedp.Run(ctx,
		// Call the login function from the script.
		chromedp.Evaluate(`usePasskey();`, &result, chromedputil.WaitForPromise),
	)
	if err != nil {
		t.Fatalf("Passkey login test failed: %v", err)
	}
	if !result.Success {
		t.Errorf("Expected login to be successful, but got: %v", result)
	}
	return result
}
