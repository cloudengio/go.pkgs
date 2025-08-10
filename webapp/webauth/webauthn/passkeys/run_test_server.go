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

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/webauth/jwtutil"
	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	serverWebauthn "github.com/go-webauthn/webauthn/webauthn"
)

var serverURL *url.URL

func init() {
	var err error
	serverURL, err = url.Parse("https://local.onyourbehalf.ai:8080")
	if err != nil {
		panic(fmt.Sprintf("Failed to parse server URL: %v", err))
	}
}

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
	ctx = ctxlog.WithLogger(ctx, logger)

	certFile := "local.pem"
	keyFile := "local-key.pem"
	devtest.NewSelfSignedCertUsingMkcert(certFile, keyFile, "local.onyourbehalf.ai")

	wa, err := serverWebauthn.New(&serverWebauthn.Config{
		RPDisplayName: "Test Passkeys",
		RPID:          serverURL.Hostname(),
		RPOrigins:     []string{serverURL.String()},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create WebAuthn instance: %v", err))
	}

	db := passkeys.NewRAMUserDatabase()
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate private key: %v", err))
	}
	signer := jwtutil.NewED25519Signer(pubKey, privKey, "pkid")
	mw := passkeys.NewJWTCookieMiddleware(signer, "pktest", time.Hour*24*30)
	w := passkeys.NewHandler(wa, db, db, mw, passkeys.WithLogger(logger))
	mime.AddExtensionType(".js", "application/javascript")

	requireResidentKey := true
	// Register the file server handler for all requests and start the server.
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./testdata")))
	mux.HandleFunc("/generate-registration-options", func(rw http.ResponseWriter, r *http.Request) {
		w.BeginRegistration(rw, r,
			protocol.MediationDefault,
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				AuthenticatorAttachment: protocol.Platform,
				RequireResidentKey:      &requireResidentKey,
				ResidentKey:             protocol.ResidentKeyRequirementRequired,
				UserVerification:        protocol.VerificationPreferred}),
		)
	})
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
		devtest.NewJSServer("generate", tsc, "passkeys.js", "passkeys-login.js").ServeJS)

	cfg, err := webapp.TLSConfigUsingCertFiles(certFile, keyFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to create TLS config: %v", err))
	}
	fmt.Printf("Starting TLS server at %s\n", serverURL.Host)
	ln, srv, err := webapp.NewTLSServer(serverURL.Host, mux, cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create TLS server: %v", err))
	}

	if err := webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second); err != nil {
		panic(fmt.Sprintf("Failed to start TLS server: %v", err))
	}
}
