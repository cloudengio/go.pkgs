// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloudeng.io/cmdutil"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/cookies"
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/webauth/jwtutil"
	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	serverWebauthn "github.com/go-webauthn/webauthn/webauthn"
)

// A minimal self-contained example of a WebAuthn server that can
// create passkeys, login in using them and verify that a user is logged
// in using a jwt cookie issued by the login.

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <host:port>")
		return
	}
	hostPort := os.Args[1]
	serverURL, err := url.Parse("https://" + hostPort)
	if err != nil {
		fmt.Printf("Failed to parse server URL: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmdutil.HandleSignals(cancel, os.Interrupt, os.Kill)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
	ctx = ctxlog.WithLogger(ctx, logger)

	certFile := "local.pem"
	keyFile := "local-key.pem"
	if err := devtest.NewSelfSignedCertUsingMkcert(certFile, keyFile, serverURL.Hostname()); err != nil {
		fmt.Printf("Failed to create self-signed certificates: %v", err)
		return
	}
	defer func() {
		os.Remove(certFile)
		os.Remove(keyFile)
	}()

	wa, err := serverWebauthn.New(&serverWebauthn.Config{
		RPDisplayName: "Test Passkeys",
		RPID:          serverURL.Hostname(),
		RPOrigins:     []string{serverURL.String()},
	})
	if err != nil {
		fmt.Printf("Failed to create WebAuthn instance: %v", err)
		return
	}

	db := passkeys.NewRAMUserDatabase()
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Printf("Failed to generate private key: %v", err)
		return
	}
	signer := jwtutil.NewED25519Signer(pubKey, privKey, "pkid")
	mw := passkeys.NewJWTCookieLoginManager(signer, "pktest", cookies.ScopeAndDuration{
		Path:     "/",
		Domain:   serverURL.Hostname(),
		Duration: time.Hour * 24 * 30,
	})
	requireResidentKey := true
	w := passkeys.NewHandler(wa, db, db, mw,
		passkeys.WithLogger(logger),
		passkeys.WithRegistrationOptions(
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				AuthenticatorAttachment: protocol.Platform,
				RequireResidentKey:      &requireResidentKey,
				ResidentKey:             protocol.ResidentKeyRequirementRequired,
				UserVerification:        protocol.VerificationPreferred,
			})),
	)
	mime.AddExtensionType(".js", "application/javascript")

	// Register the file server handler for all requests and start the server.
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./testdata")))
	mux.HandleFunc("/generate-registration-options", w.BeginRegistration)
	mux.HandleFunc("/verify-registration", w.FinishRegistration)
	mux.HandleFunc("/generate-authentication-options", w.BeginDiscoverableAuthentication)
	mux.HandleFunc("/verify-authentication", w.FinishAuthentication)
	mux.HandleFunc("/verify", w.VerifyAuthentication)

	tsc := devtest.NewTypescriptSources(
		devtest.WithTypescriptTarget("es2017"),
	)
	tsc.SetDirAndFiles("testdata", "passkeys.ts")
	mux.HandleFunc("/generate",
		devtest.NewJSServer("generate", tsc, "passkeys.js", "passkeys-create.js").ServeJS)

	mux.HandleFunc("/login",
		devtest.NewJSServer("login", tsc, "passkeys.js", "passkeys-login.js").ServeJS)

	cfg, err := webapp.TLSConfigUsingCertFiles(certFile, keyFile)
	if err != nil {
		fmt.Printf("Failed to create TLS config: %v", err)
		return
	}
	fmt.Printf("Starting TLS server at %s\n", serverURL.Host)
	ln, srv, err := webapp.NewTLSServer(serverURL.Host, mux, cfg)
	if err != nil {
		fmt.Printf("Failed to create TLS server: %v", err)
		return
	}

	if err := webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second); err != nil {
		fmt.Printf("Failed to start TLS server: %v", err)
		return
	}

	fmt.Printf("Server stopped gracefully\n")
}
