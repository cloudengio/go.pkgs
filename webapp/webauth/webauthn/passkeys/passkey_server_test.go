// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package passkeys_test

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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/webauth/jwtutil"
	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	browserWebauthn "github.com/chromedp/cdproto/webauthn"
	"github.com/chromedp/chromedp"
	serverWebauthn "github.com/go-webauthn/webauthn/webauthn"
)

var serverURL *url.URL

func init() {
	var err error
	serverURL, err = url.Parse("https://localhost:8080")
	if err != nil {
		panic(fmt.Sprintf("Failed to parse server URL: %v", err))
	}

	mime.AddExtensionType(".js", "application/javascript")

}

func runServer(ctx context.Context, tmpDir string, w *passkeys.Handler, errCh chan error) error {
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	if err := devtest.NewSelfSignedCertUsingMkcert(certFile, keyFile, "localhost"); err != nil {
		return fmt.Errorf("Failed to create self-signed certificates: %v", err)
	}
	mux := http.NewServeMux()
	//mux.HandleFunc("/ready", func(rw http.ResponseWriter, r *http.Request) {
	//	rw.WriteHeader(http.StatusOK)
	//})
	mux.HandleFunc("/generate-registration-options", w.BeginRegistration)
	mux.HandleFunc("/verify-registration", w.FinishRegistration)

	cfg, err := webapp.TLSConfigUsingCertFiles(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("Failed to create TLS config: %v", err)
	}
	fmt.Printf("Starting TLS server at %s\n", serverURL.Host)
	ln, srv, err := webapp.NewTLSServer(serverURL.Host, mux, cfg)
	if err != nil {
		return fmt.Errorf("Failed to create TLS server: %v", err)
	}
	go func() {
		errCh <- webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
	}()

	/*
		for {
			resp, _ := http.Get(fmt.Sprintf("https://%s/ready", serverURL.Host))
			if resp != nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					fmt.Println("Server is ready")
					break
				}
			}
			fmt.Printf("Waiting for server to be ready...\n")
			time.Sleep(100 * time.Millisecond)
		}*/

	return nil
}

func TestPasskeysServer(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wa, err := serverWebauthn.New(&serverWebauthn.Config{
		RPDisplayName: "Test Passkeys",
		RPID:          "localhost",
		RPOrigins:     []string{"https://localhost:8080"},
	})
	if err != nil {
		t.Fatalf("Failed to create WebAuthn instance: %v", err)
	}

	var logged strings.Builder
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	db := passkeys.NewRAMUserDatabase()
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate private key: %v", err))
	}
	signer := jwtutil.NewED25519Signer(pubKey, privKey, "pkid")
	mw := passkeys.NewJWTCookieMiddleware(
		signer, time.Minute,
		passkeys.WithLogger(logger),
	)
	w := passkeys.NewHandler(wa, db, db, mw, passkeys.WithLogger(logger))
	errCh := make(chan error, 1)
	if err := runServer(ctx, t.TempDir(), w, errCh); err != nil {
		t.Fatalf("Failed to run server: %v", err)
	}

	js, err := os.ReadFile(filepath.Join("testdata", "passkeys.js"))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}
	// Add your test cases here
	testPasskeyRegistration(t, string(js))

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Server error: %v", err)
	}

	fmt.Printf("Server logs:\n%s\n", logged.String())
}

func testPasskeyRegistration(t *testing.T, jsscript string) {
	// Standard chromedp setup
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(
		func(format string, args ...any) {
			fmt.Printf(format, args...)
		}),
	)
	defer cancel()

	// Virtual Authenticator Options for a Passkey
	authOptions := &browserWebauthn.VirtualAuthenticatorOptions{
		Protocol:            browserWebauthn.AuthenticatorProtocolCtap2,
		Transport:           browserWebauthn.AuthenticatorTransportInternal,
		HasResidentKey:      true, // Crucial for Passkeys (discoverable credentials)
		HasUserVerification: true, // Simulates biometric/PIN check
		IsUserVerified:      true, // Automatically approve consent dialogs
	}

	// --- 1. Setup Phase ---

	chromedp.ListenTarget(ctx, func(ev any) {
		// Check if the event is a log entry
		if event, ok := ev.(*log.EventEntryAdded); ok {
			fmt.Printf("log.EventEntryAdded: %v: %s\n", event.Entry.Level, event.Entry.Text)
			fmt.Printf("  - Source: %+v\n", event.Entry.Source)
			fmt.Printf("  - URL: %s\n", event.Entry.URL)
			fmt.Printf("  - Line: %d\n", event.Entry.LineNumber)
			if event.Entry.StackTrace != nil {
				fmt.Printf("  - Stack Trace: %+v\n", event.Entry.StackTrace)
			}
		} else {
			//	fmt.Printf("Event: %T\n", ev) // Print the event for debugging
		}

		if msg, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			var out strings.Builder
			for _, arg := range msg.Args {
				val, _ := arg.Value, arg.Type
				fmt.Fprintf(&out, "%v", val)
			}
			fmt.Printf("runtime.EventConsoleAPICalled: %s\n", out.String())
		}

		// You can also listen for unhandled exceptions
		if event, ok := ev.(*runtime.EventExceptionThrown); ok {
			fmt.Printf("Unhandled JS Exception: %v: %v", event.ExceptionDetails.Text, event.ExceptionDetails.Error())
			if event.ExceptionDetails.StackTrace != nil {
				fmt.Printf("  - Stack Trace: %+v\n", event.ExceptionDetails.StackTrace)
			}
			// wg.Done() // Uncomment if this is the event you want to wait for
		}
	})

	// Run a set of actions to configure the virtual authenticator.
	var authenticatorID browserWebauthn.AuthenticatorID
	if err := chromedp.Run(ctx,
		browserWebauthn.Enable(),
		// We need ActionFunc to get the ID out and into our variable.
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			authenticatorID, err = browserWebauthn.AddVirtualAuthenticator(authOptions).Do(ctx)
			return err
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Automatically simulate the "user touch" for the authenticator.
			return browserWebauthn.SetAutomaticPresenceSimulation(authenticatorID, true).Do(ctx)
		}),
	); err != nil {
		t.Fatalf("Failed to set up virtual authenticator: %v", err)
	}

	// --- 2. Defer Cleanup ---
	// This deferred function will run right before the test function returns,
	// but *before* the `defer cancel()` above closes the browser.
	defer func() {
		// Run a final set of actions to remove the authenticator.
		if err := chromedp.Run(ctx,
			browserWebauthn.RemoveVirtualAuthenticator(authenticatorID),
			browserWebauthn.Disable(),
		); err != nil {
			t.Errorf("Failed to clean up virtual authenticator: %v", err)
		}
	}()

	fmt.Printf("running chromedp now..\n")
	jsscript += `true;` // Ensure the script returns true for success.
	var pageURL string
	var scriptOK bool
	var creationResult any
	//var functions []string
	// --- 3. Test Execution Phase ---
	// Now, run the actual user-flow test.
	if err := chromedp.Run(ctx,
		chromedp.Navigate("https://localhost:8080"),
		chromedp.Evaluate(`window.location.href`, &pageURL),
		chromedp.Evaluate(jsscript, &scriptOK),
		/*
					chromedp.Evaluate(`function getGlobalFunctionNames() {
			  return Object.getOwnPropertyNames(window).filter(
			    key => typeof window[key] === 'function'
			  );
			}
			getGlobalFunctionNames();`, &functions),*/
		chromedp.Evaluate(`let r = Promise.resolve(createPasskey('test@example.com', 'test-user')); r;`, &creationResult),
		//chromedp.SendKeys("#username", "new-passkey-user"),
		//chromedp.Click("#register-with-passkey-button"),
		//chromedp.WaitVisible("#registration-successful"),
	); err != nil {
		t.Errorf("Passkey registration test failed: %v", err)
	}

	if got, want := pageURL, "https://localhost:8080/"; got != want {
		t.Errorf("Expected page URL %q, got %q", want, got)
	}
	if !scriptOK {
		t.Errorf("Expected JS evaluation to return 'ok', got %v", scriptOK)
	}
	//if !slices.Contains(functions, "createPasskey") {
	//	t.Errorf("Expected function 'createPasskey' to be defined")
	//}

	//	fmt.Printf("output ... %+v\n", output) // This will show the result of the JS evaluation.
	//	_, _ = pageURL, output

	//fmt.Printf("functions: %+v\n", functions)

	fmt.Printf("... creationResult: %v\n", creationResult)

}
