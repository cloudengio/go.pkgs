// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package chromedputil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"cloudeng.io/webapp/devtest/chromedputil"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// setupTestServer creates a simple HTTP server to serve test files.
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/test.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `
            function myTestFunction() { console.log("hello from test.js"); }
            var anotherTestFunc = () => "world";
        `)
		w.Header().Set("Content-Type", "application/javascript")
	})
	mux.HandleFunc("/non-existent.js", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mux.HandleFunc("/invalid.js", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `x=; // invalid js`)
		w.Header().Set("Content-Type", "application/javascript")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Test Page</h1></body></html>`)
	})
	return httptest.NewServer(mux)
}

// setupBrowser creates a new chromedp context and navigates to the test server.
func setupBrowser(t *testing.T, serverURL string) (context.Context, context.CancelFunc) {
	ctx, cancelA := chromedputil.ContextForCI(context.Background())
	ctx, cancelB := chromedp.NewContext(ctx)
	if err := chromedp.Run(ctx, chromedp.Navigate(serverURL)); err != nil {
		cancelA()
		t.Fatalf("failed to navigate to test server: %v", err)
	}
	return ctx, func() {
		cancelA()
		cancelB()
	}
}

func TestListGlobalFunctions(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	// Define a new global function to ensure it's picked up.
	if err := chromedp.Run(ctx, chromedp.Evaluate(`
        function aNewGlobalFunctionForTesting() {}
    `, nil)); err != nil {
		t.Fatalf("failed to define global function: %v", err)
	}

	functions, err := chromedputil.ListGlobalFunctions(ctx)
	if err != nil {
		t.Fatalf("ListGlobalFunctions failed: %v", err)
	}

	if len(functions) == 0 {
		t.Fatal("expected some global functions, but got none")
	}

	// Check for our custom function and a standard browser function.
	if !slices.Contains(functions, "aNewGlobalFunctionForTesting") {
		t.Error("expected to find 'aNewGlobalFunctionForTesting' in the list of global functions")
	}
	if !slices.Contains(functions, "fetch") {
		t.Error("expected to find standard browser function 'fetch' in the list")
	}
}

func TestWaitForPromise(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	var result int
	jsPromise := `new Promise(resolve => setTimeout(() => resolve(42), 50))`

	// Evaluate the promise using the WaitForPromise helper.
	err := chromedp.Run(ctx,
		chromedp.Evaluate(jsPromise, &result, chromedputil.WaitForPromise),
	)

	if err != nil {
		t.Fatalf("failed to evaluate promise: %v", err)
	}

	if result != 42 {
		t.Errorf("expected promise to resolve to 42, but got %d", result)
	}
}

func TestSourceScript(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	pageURL := ""
	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL),
		chromedp.Evaluate(`window.location.href`, &pageURL))
	if err != nil {
		t.Fatalf("failed to evaluate page URL: %v", err)
	}
	if got, want := pageURL, srv.URL+"/"; got != want {
		t.Errorf("expected page URL %q, got %q", want, got)
	}

	t.Run("Failed Download", func(t *testing.T) {
		// Attempt to source a script that will result in a 404.
		scriptURL := fmt.Sprintf(`"%s/non-existent.js"`, srv.URL)
		err := chromedputil.SourceScript(ctx, scriptURL)
		if err == nil {
			t.Fatal("expected SourceScript to return an error for a non-existent script, but it did not")
		}
		// The error from a rejected promise in chromedp includes the rejection reason.
		if !strings.Contains(err.Error(), "Failed to load script") {
			t.Errorf("expected error message to indicate script load failure, but got: %v", err)
		}
	})

	t.Run("Failed Parsing of downloaded JS", func(t *testing.T) {
		// Attempt to source a script that will result in a 404.
		scriptURL := fmt.Sprintf(`"%s/invalid.js"`, srv.URL)
		err := chromedputil.SourceScript(ctx, scriptURL)
		if err == nil {
			t.Fatal("expected SourceScript to return an error for an invalid script, but it did not")
		}
		// The error from a rejected promise in chromedp includes the rejection reason.
		if !strings.Contains(err.Error(), "SyntaxError: Unexpected token ';'") {
			t.Errorf("expected error message to indicate script load failure, but got: %v", err)
		}
	})

	t.Run("Successful Load", func(t *testing.T) {
		// Source the test script from the server.
		scriptURL := fmt.Sprintf(`"%s/test.js"`, srv.URL)
		if err := chromedputil.SourceScript(ctx, scriptURL); err != nil {
			t.Fatalf("SourceScript failed: %v", err)
		}

		// Verify that the functions from the script are now defined.
		functions, err := chromedputil.ListGlobalFunctions(ctx)
		if err != nil {
			t.Fatalf("ListGlobalFunctions failed: %v", err)
		}
		if !slices.Contains(functions, "myTestFunction") {
			t.Error("expected 'myTestFunction' to be defined after sourcing script")
		}
		if !slices.Contains(functions, "anotherTestFunc") {
			t.Error("expected 'anotherTestFunc' to be defined after sourcing script")
		}
	})

}

func TestGetRemoteObject(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	// Define a global object in the browser's context for testing.
	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL),
		chromedp.Evaluate(`
        const myTestObject = { key: "value" };
    `, nil))
	if err != nil {
		t.Fatalf("failed to define global object: %v", err)
	}

	t.Run("Successful Get", func(t *testing.T) {
		obj, err := chromedputil.GetRemoteObjectRef(ctx, "myTestObject")
		if err != nil {
			t.Fatalf("GetRemoteObject failed: %v", err)
		}
		if obj == nil {
			t.Fatal("GetRemoteObject returned a nil object")
		}
		if obj.ObjectID == "" {
			t.Error("expected a non-empty ObjectID")
		}
		if obj.Type != "object" {
			t.Errorf("expected object type 'object', got %q", obj.Type)
		}
	})

	t.Run("Object Not Found", func(t *testing.T) {
		_, err := chromedputil.GetRemoteObjectRef(ctx, "nonExistentObject")
		if err == nil {
			t.Fatal("expected an error when getting a non-existent object, but got nil")
		}
		// The underlying error is a ReferenceError from JS, which chromedp surfaces.
		if !strings.Contains(err.Error(), "ReferenceError") {
			t.Errorf("expected error to be a ReferenceError, but got: %v", err)
		}
	})
}

func TestGetRemoteObjectValue(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	// Define a global object to test against.
	const testObjJS = `const myValueObject = { name: "test", id: 123, nested: { active: true } };`

	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL),
		chromedp.Evaluate(testObjJS, nil))
	if err != nil {
		t.Fatalf("failed to define global object: %v", err)
	}

	// First, get a handle to the remote object.
	handle, err := chromedputil.GetRemoteObjectRef(ctx, "myValueObject")
	if err != nil {
		t.Fatalf("failed to get initial handle: %v", err)
	}

	t.Run("Successful Get Value", func(t *testing.T) {
		// Now, use the handle to get the object's actual value.
		valueObject, value, err := chromedputil.GetRemoteObjectValueJSON(ctx, handle)
		if err != nil {
			t.Fatalf("GetRemoteObjectValue failed: %v", err)
		}
		if valueObject == nil {
			t.Fatal("GetRemoteObjectValue returned a nil object")
		}
		if value == nil {
			t.Fatal("expected the returned object's Value field to be populated")
		}

		// Unmarshal the JSON value and verify its contents.
		var result map[string]any
		if err := json.Unmarshal(value, &result); err != nil {
			t.Fatalf("failed to unmarshal object value: %v", err)
		}

		expected := map[string]any{
			"name": "test",
			"id":   float64(123), // JSON numbers are unmarshaled as float64
			"nested": map[string]any{
				"active": true,
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("object value mismatch:\ngot:  %v\nwant: %v", result, expected)
		}
	})

	t.Run("Invalid ObjectID", func(t *testing.T) {
		invalidHandle := &runtime.RemoteObject{
			ObjectID: "invalid-id-12345",
		}
		_, _, err := chromedputil.GetRemoteObjectValueJSON(ctx, invalidHandle)
		if err == nil {
			t.Fatal("expected an error for an invalid ObjectID, but got nil")
		}
		if !strings.Contains(err.Error(), "Invalid remote object") {
			t.Errorf("expected error about missing object, but got: %v", err)
		}
	})
}

func TestPlatformObjects(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	testCases := []struct {
		name           string
		jsExpression   string
		expectedType   string
		expectedClass  string
		waitForPromise bool
	}{
		{
			name:           "Response Object",
			expectedType:   "PlatformObject",
			jsExpression:   `fetch("/")`,
			expectedClass:  "Response",
			waitForPromise: true,
		},
		{
			name:          "Promise Object",
			expectedType:  "PlatformObject",
			jsExpression:  `fetch("/")`,
			expectedClass: "Promise",
		},
		{
			name:          "Error Object",
			expectedType:  "PlatformObject",
			jsExpression:  `new Error("test error")`,
			expectedClass: "Error",
		},
		{
			name:          "Document Object",
			expectedType:  "Document",
			jsExpression:  `document`,
			expectedClass: "", // no class for Document
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Get a handle to the platform object.
			var handle *runtime.RemoteObject

			err := chromedp.Run(ctx,
				chromedp.Evaluate(tc.jsExpression, &handle,
					func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
						p.AwaitPromise = tc.waitForPromise
						return p
					}))

			if err != nil {
				t.Fatalf("failed to get handle for %q: %v", tc.name, err)
			}

			// Step 2: Get its value, which should be a safe-cloned JSON representation.
			valueObject, value, err := chromedputil.GetRemoteObjectValueJSON(ctx, handle)
			if err != nil {
				t.Fatalf("GetRemoteObjectValueJSON failed for %q: %v", tc.name, err)
			}

			// Step 3: Verify it's identified as a platform object.
			if !chromedputil.IsPlatformObject(valueObject) {
				t.Errorf("expected IsPlatformObject to be true for %q, but it was false", tc.name)
			}

			// Step 4: Unmarshal the JSON and check its contents.
			platformInfo := struct {
				Type      string `json:"_type"`
				ClassName string `json:"className"`
			}{}
			if err := json.Unmarshal(value, &platformInfo); err != nil {
				t.Logf("failed to unmarshal platform object info: value: %v: err: %v", value, err)
				t.Fatalf("failed to unmarshal platform object info: %v", err)
			}

			if platformInfo.Type != tc.expectedType {
				t.Errorf("expected _type to be %q, got %q", tc.expectedType, platformInfo.Type)
			}
			if platformInfo.ClassName != tc.expectedClass {
				t.Errorf("expected className to be %q, got %q", tc.expectedClass, platformInfo.ClassName)
			}
		})
	}
}

func TestConsoleArgsAsJSON(t *testing.T) { //nolint:gocyclo
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	consoleEventCh := make(chan *runtime.EventConsoleAPICalled, 1)
	chromedputil.Listen(ctx, chromedputil.NewEventConsoleHandler(consoleEventCh))

	// This JS will log a variety of object types to the console.
	const jsLogComplexObjects = `
        const simpleObj = { name: "test", value: 42 };
        const nestedObj = { a: 1, b: { c: "nested" } };
        const anArray = [1, "two", false];
        const aPromise = Promise.resolve({ status: "ok" });
        const aResponse = fetch("/"); // fetch returns a promise that resolves to a Response
        
        Promise.all([aPromise, aResponse]).then(([p, r]) => {
            console.log(
                "a string",
                123,
                true,
                simpleObj,
                nestedObj,
                anArray,
                p, // The resolved value of the promise
                r  // The Response object
            );
        });
    `

	// We need to wait for the console.log inside the promise to execute.
	err := chromedp.Run(ctx,
		chromedp.Evaluate(jsLogComplexObjects, nil, chromedputil.WaitForPromise),
	)
	if err != nil {
		t.Fatalf("failed to execute logging script: %v", err)
	}

	var event *runtime.EventConsoleAPICalled
	select {
	case event = <-consoleEventCh:
		// Event received.
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for console event")
	}

	jsonArgs, err := chromedputil.ConsoleArgsAsJSON(ctx, event)
	if err != nil {
		t.Fatalf("ConsoleArgsAsJSON failed: %v", err)
	}

	if got, want := len(jsonArgs), 8; got != want {
		t.Fatalf("got %d, want %d", got, want)
	}

	// Define checks for each logged argument.
	checks := []func(t *testing.T, data []byte){
		// 1. String
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `"a string"`; string(obj.Value) != want {
				t.Errorf("got %s, want %s", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 2. Number
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `123`; string(obj.Value) != want {
				t.Errorf("got %s, want %s", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 3. Boolean
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `true`; string(obj.Value) != want {
				t.Errorf("got %s, want %s", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 4. Simple Object
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `{"name":"test","value":42}`; string(obj.Value) != want {
				t.Errorf("got %v, want %v", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 5. Nested Object
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `{"a":1,"b":{"c":"nested"}}`; string(obj.Value) != want {
				t.Errorf("got %s, want %s", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 6. Array
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `[1,"two",false]`; string(obj.Value) != want {
				t.Errorf("got %s, want %s", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 7. Resolved Promise Value
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			if want := `{"status":"ok"}`; string(obj.Value) != want {
				t.Errorf("got %s, want %s", obj.Value, want)
			}
			if chromedputil.IsPlatformObject(&obj) {
				t.Errorf("did not expect a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
		// 8. Response Object
		func(t *testing.T, data []byte) {
			var obj runtime.RemoteObject
			if err := json.Unmarshal(data, &obj); err != nil {
				t.Fatal(err)
			}
			// For complex browser-native objects like Response, we can't get a simple
			// JSON value. We verify that we get a handle with the correct type.
			if obj.Type != "object" || obj.ClassName != "Response" {
				t.Errorf("expected a Response object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
			if !chromedputil.IsPlatformObject(&obj) {
				t.Errorf("expected a platform object, but got type %q and class %q", obj.Type, obj.ClassName)
			}
		},
	}

	for i, arg := range jsonArgs {
		t.Run(fmt.Sprintf("Argument %d", i), func(t *testing.T) {
			checks[i](t, arg)
		})
	}
}

func TestRunLoggingListener(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	// Capture slog output
	var logBuf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	// Run the listener with console and exception logging enabled.
	chromedputil.RunLoggingListener(ctx, logger,
		chromedputil.WithConsoleLogging(),
		chromedputil.WithExceptionLogging(),
	)

	// Generate a console log and an exception in the browser.
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`console.log("hello from the test");`, nil),
	)
	// We expect an error from the second evaluate, so we don't fail the test on it.
	if err != nil {
		t.Logf("chromedp.Run returned an expected error: %v", err)
	}

	// generate an exception..
	scriptURL := fmt.Sprintf(`"%s/invalid.js"`, srv.URL)
	err = chromedputil.SourceScript(ctx, scriptURL)
	if err == nil {
		t.Fatal("expected SourceScript to return an error for an invalid script, but it did not")
	}

	// Give the listener a moment to process the events.
	time.Sleep(200 * time.Millisecond)

	// Stop the listener and capture the output.
	cancel()

	logOutput := logBuf.String()

	// Verify the console log was captured and printed to stderr.
	expectedConsoleOut := `hello from the test`
	if !strings.Contains(logOutput, expectedConsoleOut) {
		t.Errorf("expected stderr to contain %q, but got:\n%s", expectedConsoleOut, logOutput)
	}

	// Verify the exception was captured and logged by slog.
	if !strings.Contains(logOutput, "Exception thrown") {
		t.Errorf("expected log output to contain 'Exception thrown', but got:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "SyntaxError: Unexpected token ';'") {
		t.Errorf("expected log output to contain 'SyntaxError: Unexpected token ';'', but got:\n%s", logOutput)
	}
}
