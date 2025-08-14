// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

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
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/webauth/jwtutil"
	"cloudeng.io/webapp/webauth/webauthn/passkeys"
	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	browserWebauthn "github.com/chromedp/cdproto/webauthn"
	"github.com/chromedp/chromedp"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	serverWebauthn "github.com/go-webauthn/webauthn/webauthn"
)

var serverURL *url.URL

func init() {
	var err error
	serverURL, err = url.Parse("https://localhost:8081")
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
	ln, srv, err := webapp.NewTLSServer(serverURL.Host, mux, cfg)
	if err != nil {
		return fmt.Errorf("failed to create TLS server: %v", err)
	}

	go func() {
		errCh <- webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
	}()

	return nil
}

func TestPasskeysServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wa, err := serverWebauthn.New(&serverWebauthn.Config{
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
	mw := passkeys.NewJWTCookieMiddleware(
		signer, "localhost", time.Minute)
	w := passkeys.NewHandler(wa, db, db, mw,
		passkeys.WithLogger(logger),
		passkeys.WithRegistrationOptions(
			webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
				AuthenticatorAttachment: protocol.Platform,
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
	defer func() {
		if err := chromedp.Run(ctx, browserWebauthn.RemoveVirtualAuthenticator(authenticatorID)); err != nil {
			t.Errorf("Failed to remove virtual authenticator: %v", err)
		}
	}()

	// Run tests for registration and login.
	testPasskeyRegistration(ctx, t)
	testPasskeyLogin(ctx, t)

	cancel()
	if err := <-errCh; err != nil {
		// http.ErrServerClosed is the expected error on graceful shutdown.
		if err != http.ErrServerClosed {
			t.Fatalf("Server error: %v", err)
		}
	}
	t.Logf("Server logs:\n%s\n", logged.String())
}

func pars(objID runtime.RemoteObjectID) *runtime.GetPropertiesParams {
	return &runtime.GetPropertiesParams{
		ObjectID:                 objID,
		OwnProperties:            true,
		AccessorPropertiesOnly:   false,
		GeneratePreview:          true,
		NonIndexedPropertiesOnly: false,
	}
}

/*
func printObject(ctx context.Context, arg *runtime.RemoteObject, indent string) (string, error) {
	var sb strings.Builder
	switch {
	case arg.Value != nil:
		fmt.Fprintf(&sb, "%s%02d: %s\n", indent, pos, arg.Value)
		return sb.String(), nil
	case arg.ObjectID != "":
		props, _, _, _, err := pars(arg.ObjectID).Do(ctx)
		if err != nil {
			fmt.Fprintf(&sb, "%s%02d: %s\n", indent, pos, err)
			return sb.String(), fmt.Errorf("Error getting properties: %w\n", err)
		}
		for npos, prop := range props {
			fmt.Fprintf(&sb, "%s%02d: %v: __%v++\n", indent, npos, prop.Name, prop.Value.Description)
			out, err := printObject(ctx, prop.Value, npos, indent+"  ")
			if err != nil {
				return out, err
			}
			sb.WriteString(out)
		}
		//fmt.Fprintf(&sb, "%s%02d: End of object %v\n", indent, pos, arg.ObjectID)
	default:
		fmt.Fprintf(&sb, "%s%02d: (unknown)\n", indent, pos)
	}
	return sb.String(), nil
}*/
/*
func printObject(ctx context.Context, objectID runtime.RemoteObjectID, indent string) (string, error) {
	var sb strings.Builder
	props, _, _, _, err := pars(objectID).Do(ctx)
	if err != nil {
		fmt.Fprintf(&sb, "%s: %s\n", indent, err)
		return sb.String(), fmt.Errorf("Error getting properties: %w\n", err)
	}
	for _, prop := range props {
		switch {
		case prop.Value.Value != nil:
			fmt.Fprintf(&sb, "1 %s: %s: %s\n", indent, prop.Name, prop.Value.Value)
		case prop.Value.ObjectID != "":
			out, err := printObject(ctx, prop.Value.ObjectID, indent)
			if err != nil {
				return out, fmt.Errorf("Error printing object: %w\n", err)
			}
			fmt.Fprintf(&sb, "_______ %s: %v: %v\n", indent, prop.Name, prop.Value.Description)
			sb.WriteString(out)
		default:
			fmt.Fprintf(&sb, "%s%s: (unknown)\n", indent, prop.Name)
		}
	}
	return sb.String(), nil
}

func getRemoteObject(ctx context.Context, objectID runtime.RemoteObjectID) (*runtime.RemoteObject, error) {
	res, exp, err := runtime.CallFunctionOn(`function() { return this; }`).
		WithObjectID(objectID).
		WithReturnByValue(true).
		/*WithSerializationOptions(&runtime.SerializationOptions{
			Serialization: runtime.SerializationOptionsSerializationJSON,
			//			MaxDepth:      10,
		}).*/ /*
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if exp != nil {
		fmt.Printf("Exception: %+v\n", exp)
		return nil, fmt.Errorf("Exception: %v (%+v)", exp.Text, exp)
	}
	return res, nil
	// b, _ := json.MarshalIndent(res.DeepSerializedValue.Value, "", "  ")
	// fmt.Println("Deep JSON:", string(b))
	// _ = res
}

func printConsoleArgs(ctx context.Context, args []*runtime.RemoteObject) (string, error) {
	var sb strings.Builder
	indent := "  "
	for pos, arg := range args {
		switch {
		case arg.Value != nil:
			fmt.Fprintf(&sb, "%s%02d: %s\n", indent, pos, arg.Value)
		case arg.ObjectID != "":
			if arg.ClassName == "Response" {
				// Handle Response objects differently
				fmt.Fprintf(&sb, "%s%02d: <Server-Response>\n", indent, pos)
				continue
			} /*
				out, err := printObject(ctx, arg.ObjectID, indent)
				if err != nil {
					return out, fmt.Errorf("Error printing object: %w\n", err)
				}*/ /*
	obj, err := getRemoteObject(ctx, arg.ObjectID)
	if err != nil {
		return "", fmt.Errorf("Error getting remote object: %w\n", err)
	}
	var out strings.Builder
	if err := json.NewEncoder(&out).Encode(obj); err != nil {
		return "", fmt.Errorf("Error encoding JSON: %w\n", err)
	}
	fmt.Fprintf(&sb, "%s%02d: %s\n", indent, pos, out.String())

	/*
		props, _, _, _, err := pars(arg.ObjectID).Do(ctx)
		fmt.Fprintf(&sb, "%s%02d: %v (%+v) [%+v]\n", indent, pos, out, arg, props)
		_ = err*/ /*
			//xx, err := getjson(ctx, execCtxID, arg)
			//fmt.Fprintf(os.Stderr, "XXXX (%v) %s\n", err, xx)
		default:
			fmt.Fprintf(&sb, "%s%02d: (unknown)\n", indent, pos)
		}
	}
	return sb.String(), nil
}*/

/*
// This JavaScript function stringifies an object while safely handling circular references.
const safeStringify = `function() {
    const cache = new Set();
    return JSON.stringify(this, (key, value) => {
        if (typeof value === 'object' && value !== null) {
            if (cache.has(value)) {
                // Circular reference found, discard key
                return "[Circular Reference]";
            }
            // Store value in our collection
            cache.add(value);
        }
        return value;
    });
}`

func getjson(ctx context.Context, execCtxID runtime.ExecutionContextID, remoteObject *runtime.RemoteObject) (string, error) {

	const funcDeclaration = `function() { return JSON.stringify(this); }`

	params := runtime.CallFunctionOn(safeStringify).
		//WithExecutionContextID(execCtxID).
		WithObjectID(remoteObject.ObjectID)
		//WithSerializationOptions(&runtime.SerializationOptions{
		//	Serialization: runtime.SerializationOptionsSerializationJSON,
		//})
	//params.FunctionDeclaration = funcDeclaration

	// Execute the call
	res, exp, err := params.Do(ctx)
	if err != nil {
		return string(remoteObject.ObjectID), err
	}
	if exp != nil {
		fmt.Printf("Exception: %+v\n", exp)
		return "BB", exp
	}

	// The deeply serialized value is in res.Result.Value
	var deepValue struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	}
	fmt.Printf(">>>>>>>>>>> %+v\n", deepValue)
	if err := json.Unmarshal(res.Value, &deepValue); err != nil {
		return "CC", err
	}

	// The actual JSON data is inside the 'value' field.
	return string(deepValue.Value), nil
}*/

func setupBrowser(t *testing.T) (context.Context, context.CancelFunc, browserWebauthn.AuthenticatorID) {
	t.Helper()
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(t.Logf))

	// Listen for and log any console messages from the browser.
	chromedp.ListenTarget(ctx, func(ev any) {
		if msg, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			go func() {
				chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
					out, err := printConsoleArgs(ctx, msg.Args)
					fmt.Printf("...chromedp: console:\n%s\n", out)
					_ = out
					return err
				}))
			}()
		}
		if event, ok := ev.(*log.EventEntryAdded); ok {
			t.Logf("...chromedp: event: %v: %s\n", event.Entry.URL, event.Entry.Text)
			if event.Entry.StackTrace != nil {
				t.Logf("  - Stack Trace: %+v\n", event.Entry.StackTrace)
			}
		}
		if event, ok := ev.(*runtime.EventExceptionThrown); ok {
			t.Logf("...chromedp: Unhandled JS Exception: %v: %v", event.ExceptionDetails.Text, event.ExceptionDetails.Error())
			if event.ExceptionDetails.StackTrace != nil {
				t.Logf("  - Stack Trace: %+v\n", event.ExceptionDetails.StackTrace)
			}
		}
		//fmt.Printf("...chromedp: %T: %+v\n", ev, ev)
	})

	authOptions := &browserWebauthn.VirtualAuthenticatorOptions{
		Protocol:            browserWebauthn.AuthenticatorProtocolCtap2,
		Transport:           browserWebauthn.AuthenticatorTransportInternal,
		HasResidentKey:      true,
		HasUserVerification: true,
		IsUserVerified:      true,
	}

	var authenticatorID browserWebauthn.AuthenticatorID
	if err := chromedp.Run(ctx,
		browserWebauthn.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			authenticatorID, err = browserWebauthn.AddVirtualAuthenticator(authOptions).Do(ctx)
			return err
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return browserWebauthn.SetAutomaticPresenceSimulation(authenticatorID, true).Do(ctx)
		}),
	); err != nil {
		cancel()
		t.Fatalf("Failed to set up virtual authenticator: %v", err)
	}

	//if err := loadScript(ctx, t, "/content/passkeys.js"); err != nil {
	//	cancel()
	//	t.Fatalf("Failed to load script: %v", err)
	//}

	return ctx, cancel, authenticatorID
}

func waitForPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

func testPasskeyRegistration(ctx context.Context, t *testing.T) {
	result := struct {
		UserHandle  string `json:"user_handle"`
		PublicKeyID string `json:"public_key_id"`
		Email       string `json:"email"`
		Exception   string `json:"exception"`
		Error       string `json:"error"`
	}{}
	err := chromedp.Run(ctx,
		chromedp.Navigate(serverURL.String()+"/generate"),
		// Call the registration function from the script.
		chromedp.Evaluate(`createPasskey('test@example.com', 'Test User').then((result) => { return result; });`, &result, waitForPromise),
	)
	t.Logf("Registration result: %+v", result)
	if err != nil {
		t.Fatalf("Passkey registration test failed: %v", err)
	}
	fmt.Printf(">>>>>>>>>>> %+v\n", result)
	t.Fail()

}

func testPasskeyLogin(ctx context.Context, t *testing.T) {
	ctx, cancel, authenticatorID := setupBrowser(t)
	defer cancel()
	defer func() {
		if err := chromedp.Run(ctx, browserWebauthn.RemoveVirtualAuthenticator(authenticatorID)); err != nil {
			t.Errorf("Failed to remove virtual authenticator: %v", err)
		}
	}()

	var result string
	err := chromedp.Run(ctx,
		chromedp.Navigate(serverURL.String()),
		// Call the login function from the script.
		chromedp.Evaluate(`loginWithPasskey()`, &result),
	)

	if err != nil {
		t.Fatalf("Passkey login test failed: %v", err)
	}
	if result != "login successful" {
		t.Errorf("Expected login to be successful, but got: %v", result)
	}
}
