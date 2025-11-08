// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package chromedputil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devtest/chromedputil"
	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func setupTestEnvironment(t *testing.T) (context.Context, context.CancelFunc, string) {
	// Setup a test server that will trigger various browser events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api-endpoint" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message": "Hello from API endpoint"}`)) //nolint:errcheck
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`
            <html>
                <head><title>Test Page</title></head>
                <body>
                    <h1>Testing chromedputil</h1>
                    <script>
                        // This will be detected by console logging
                        console.log("Hello from test page", {key: "value"});
                        
                        // This will trigger network events
                        fetch("/api-endpoint").then(r => console.log("Fetch completed"));

                        // This will trigger an exception event
                        throw new Error("Planned test error");
                     </script>                </body>
            </html>
        `))
	}))

	t.Cleanup(func() { server.Close() })

	extraExecOpts := debuggingExecOpts(false)

	ctx, cancel := chromedputil.WithContextForCI(context.Background(),
		extraExecOpts,
		debuggingCtxOpts(t, false)...,
	)

	return ctx, cancel, server.URL
}

func debuggingExecOpts(debug bool) []chromedp.ExecAllocatorOption {
	var extraExecOpts []chromedp.ExecAllocatorOption
	if debug {
		extraExecOpts = append(extraExecOpts, chromedp.CombinedOutput(&chromeWriter{os.Stderr}))
		extraExecOpts = append(extraExecOpts, chromedputil.AllocatorLoggingWithLevel(1)...)
	}
	return extraExecOpts
}

func debuggingCtxOpts(t *testing.T, debug bool) []chromedp.ContextOption {
	var ctxOpts []chromedp.ContextOption
	if debug {
		ctxOpts = append(ctxOpts,
			chromedp.WithBrowserOption(
				chromedp.WithBrowserDebugf(t.Logf),
				chromedp.WithBrowserLogf(t.Logf),
				chromedp.WithBrowserErrorf(t.Logf)),
			chromedp.WithLogf(t.Logf),
			chromedp.WithDebugf(t.Logf),
			chromedp.WithErrorf(t.Logf))
	}
	return ctxOpts
}

type chromeWriter struct{ io.Writer }

func (w chromeWriter) Write(p []byte) (n int, err error) {
	o := append([]byte("chrome(output): "), p...)
	_, err = w.Writer.Write(o)
	return len(p), err
}

func TestListen(t *testing.T) {
	ctx, cancel, serverURL := setupTestEnvironment(t)
	defer cancel()

	// Create channels to receive different event types
	consoleCh := make(chan *runtime.EventConsoleAPICalled, 10)
	exceptionCh := make(chan *runtime.EventExceptionThrown, 10)

	// Set up event handlers for console events and exceptions
	chromedputil.Listen(ctx,
		chromedputil.NewListenHandler(consoleCh),
		chromedputil.NewListenHandler(exceptionCh),
	)

	wctx, wcancel := context.WithTimeout(ctx, time.Minute)
	defer wcancel()

	if err := chromedp.Run(wctx,
		chromedp.Navigate(serverURL),
	); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Verify console events were captured
	select {
	case event := <-consoleCh:
		if event.Type != "log" {
			t.Errorf("Expected log event, got %s", event.Type)
		}
		if len(event.Args) < 1 {
			t.Errorf("Expected at least one console argument")
		}
	case <-time.After(3 * time.Second):
		t.Error("Timed out waiting for console event")
	}

	// We may or may not get an exception event depending on timing,
	// but we don't want to fail the test if we don't
	select {
	case <-exceptionCh:
		// Success - exception was captured
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for exception event")
	}

}

// 1. String
func testString(t *testing.T, data []byte) {
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
}

// 2. Number
func testNumber(t *testing.T, data []byte) {
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
}

// 3. Boolean
func testBoolean(t *testing.T, data []byte) {
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
}

// 4. Simple Object
func testSimpleObject(t *testing.T, data []byte) {
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
}

// 5. Nested Object
func testNestedObject(t *testing.T, data []byte) {
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
}

// 6. Array
func testArray(t *testing.T, data []byte) {
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
}

// 7. Resolved Promise Value
func testResolvedPromise(t *testing.T, data []byte) {
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
}

// 8. Response Object
func testResponseObject(t *testing.T, data []byte) {
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
}

func consoleArgsChecks() []func(t *testing.T, data []byte) {
	return []func(t *testing.T, data []byte){
		testString,
		testNumber,
		testBoolean,
		testSimpleObject,
		testNestedObject,
		testArray,
		testResolvedPromise,
		testResponseObject,
	}

}

func TestConsoleArgsAsJSONGemini(t *testing.T) {
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	consoleEventCh := make(chan *runtime.EventConsoleAPICalled, 1)
	chromedputil.Listen(ctx, chromedputil.NewListenHandler(consoleEventCh))

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
	checks := consoleArgsChecks()

	for i, arg := range jsonArgs {
		t.Run(fmt.Sprintf("Argument %d", i), func(t *testing.T) {
			checks[i](t, arg)
		})
	}
}

func TestConsoleArgsAsJSONClaude(t *testing.T) {
	ctx, cancel, _ := setupTestEnvironment(t)
	defer cancel()

	// Create a channel to receive console events
	consoleCh := make(chan *runtime.EventConsoleAPICalled, 10)

	// Set up an event handler for console events
	chromedputil.Listen(ctx, chromedputil.NewListenHandler(consoleCh))

	// Execute JavaScript that logs various types of values
	if err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(`
            console.log(
                "string value", 
                123, 
                true, 
                {name: "test object", nested: {value: 42}},
                ["array", "items"]
            );
        `, nil),
	); err != nil {
		t.Fatalf("Failed to execute JavaScript: %v", err)
	}

	// Wait for the console event
	var event *runtime.EventConsoleAPICalled
	select {
	case event = <-consoleCh:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for console event")
	}

	// Convert console args to JSON
	jsonData, err := chromedputil.ConsoleArgsAsJSON(ctx, event)
	if err != nil {
		t.Fatalf("ConsoleArgsAsJSON failed: %v", err)
	}

	// Check we got the expected number of arguments
	if len(jsonData) != 5 {
		t.Errorf("Expected 5 JSON values, got %d", len(jsonData))
	}

	// Check the contents of the first argument (string)
	if !bytes.Contains(jsonData[0], []byte(`"value"`)) {
		t.Errorf("First argument should contain a string value: %s", jsonData[0])
	}

	// Check the contents of the fourth argument (object)
	if !bytes.Contains(jsonData[3], []byte(`"name"`)) || !bytes.Contains(jsonData[3], []byte(`"nested"`)) {
		t.Errorf("Fourth argument should contain object properties: %s", jsonData[3])
	}

	// Check the contents of the fifth argument (array)
	if !bytes.Contains(jsonData[4], []byte(`"array"`)) || !bytes.Contains(jsonData[4], []byte(`"items"`)) {
		t.Errorf("Fifth argument should contain array items: %s", jsonData[4])
	}
}

func TestRunLoggingListenerClaude(t *testing.T) {
	ctx, cancel, serverURL := setupTestEnvironment(t)
	defer cancel()

	// Create a buffered writer to capture log output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	// Start the logging listener with all event types enabled
	listenerCtx, cancelListener := context.WithCancel(ctx)
	doneCh := chromedputil.RunLoggingListener(listenerCtx, logger,
		chromedputil.WithConsoleLogging(),
		chromedputil.WithExceptionLogging(),
		chromedputil.WithNetworkLogging(),
		chromedputil.WithEventEntryLogging(),
		chromedputil.WithAnyEventLogging(),
	)

	if err := webapp.WaitForURLs(ctx, time.Second, serverURL); err != nil {
		t.Fatalf("Failed to wait for server URL: %v", err)
	}

	// Navigate to the test page which will trigger various events
	wctx, wcancel := context.WithTimeout(ctx, time.Minute)
	defer wcancel()
	if err := chromedp.Run(wctx, chromedp.Navigate(serverURL)); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Give time for events to be processed
	time.Sleep(1 * time.Second)

	// Shutdown the listener
	cancelListener()

	// Wait for the listener to finish
	select {
	case <-doneCh:
		// Success - listener has terminated
		t.Logf("Listener terminated successfully")
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for listener to terminate")
	}

	// Check that logs were captured
	logs := logBuf.String()
	fmt.Printf("...Captured logs:\n%s\n", logs)
	t.Logf("...Captured logs:\n%s", logs)

	// Should have captured console logs
	if !strings.Contains(logs, "Console API called") {
		t.Error("No console logs were captured")
	}

	// Should have captured a network request
	if !strings.Contains(logs, "Network request received") {
		t.Error("No network request logs were captured")
	}

	// Should have captured a network response
	if !strings.Contains(logs, "Network response received") {
		t.Error("No network response logs were captured")
	}

	// Should have captured an exception
	if !strings.Contains(logs, "Exception thrown") {
		t.Error("No exception logs were captured")
	}

	if !strings.Contains(logs, "Log entry added") {
		t.Error("No log entry logs were captured")
	}

}

func testNewListenHandler(ctx context.Context, t *testing.T, event any, expected bool) { //nolint:gocyclo
	t.Helper()

	// Create a channel for the specific event type
	switch evt := event.(type) {
	case *runtime.EventConsoleAPICalled:
		ch := make(chan *runtime.EventConsoleAPICalled, 1)
		handler := chromedputil.NewListenHandler(ch)

		// Call the handler with the event
		result := handler(ctx, evt)
		if result != expected {
			t.Errorf("Handler returned %v, expected %v", result, expected)
		}

		if expected {
			select {
			case <-ch:
				// Success - event was sent to channel
			case <-time.After(100 * time.Millisecond):
				t.Error("Event was not sent to channel")
			}
		}

	case *runtime.EventExceptionThrown:
		ch := make(chan *runtime.EventExceptionThrown, 1)
		handler := chromedputil.NewListenHandler(ch)

		// Call the handler with the event
		result := handler(ctx, evt)
		if result != expected {
			t.Errorf("Handler returned %v, expected %v", result, expected)
		}

		if expected {
			select {
			case <-ch:
				// Success - event was sent to channel
			case <-time.After(100 * time.Millisecond):
				t.Error("Event was not sent to channel")
			}
		}

	case *log.EventEntryAdded:
		ch := make(chan *log.EventEntryAdded, 1)
		handler := chromedputil.NewListenHandler(ch)

		// Call the handler with the event
		result := handler(ctx, evt)
		if result != expected {
			t.Errorf("Handler returned %v, expected %v", result, expected)
		}

		if expected {
			select {
			case <-ch:
				// Success - event was sent to channel
			case <-time.After(100 * time.Millisecond):
				t.Error("Event was not sent to channel")
			}
		}

	case string:
		ch := make(chan string, 1)
		handler := chromedputil.NewListenHandler(ch)

		// Call the handler with a mismatched event type
		result := handler(ctx, &runtime.EventConsoleAPICalled{})
		if result != false {
			t.Error("Handler should return false for mismatched event types")
		}
	}
}

func TestNewListenHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with different event types
	testCases := []struct {
		name     string
		event    any
		expected bool
	}{
		{
			name:     "Console Event",
			event:    &runtime.EventConsoleAPICalled{},
			expected: true,
		},
		{
			name:     "Exception Event",
			event:    &runtime.EventExceptionThrown{},
			expected: true,
		},
		{
			name:     "Log Event",
			event:    &log.EventEntryAdded{},
			expected: true,
		},
		{
			name:     "Wrong Event Type",
			event:    "not an event",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testNewListenHandler(ctx, t, tc.event, tc.expected)
		})
	}
}

// Test handling of network events specifically
func TestNetworkEvents(t *testing.T) {
	ctx, cancel, serverURL := setupTestEnvironment(t)
	defer cancel()

	// Create channels for network events
	requestCh := make(chan *network.EventRequestWillBeSent, 10)
	responseCh := make(chan *network.EventResponseReceived, 10)

	// Set up event handlers
	chromedputil.Listen(ctx,
		chromedputil.NewListenHandler(requestCh),
		chromedputil.NewListenHandler(responseCh),
	)

	// Enable network event collection
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		t.Fatalf("Failed to enable network: %v", err)
	}

	// Navigate to the test page
	if err := chromedp.Run(ctx, chromedp.Navigate(serverURL)); err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Wait for and verify request event
	var requestEvent *network.EventRequestWillBeSent
	select {
	case requestEvent = <-requestCh:
		if !strings.Contains(requestEvent.Request.URL, serverURL) {
			t.Errorf("Request URL %s doesn't contain server URL %s", requestEvent.Request.URL, serverURL)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for request event")
	}

	// Wait for and verify response event
	var responseEvent *network.EventResponseReceived
	select {
	case responseEvent = <-responseCh:
		if responseEvent.Response.Status != 200 {
			t.Errorf("Expected 200 status, got %d", responseEvent.Response.Status)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for response event")
	}
}

func TestRunLoggingListenerEvaluate(t *testing.T) {
	// Generate events using chromdp.Evaluate rather than page load
	// of html that contains a js script.
	srv := setupTestServer()
	defer srv.Close()

	ctx, cancel := setupBrowser(t, srv.URL)
	defer cancel()

	// Capture slog output
	var logBuf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	// Run the listener with console and exception logging enabled.
	doneCh := chromedputil.RunLoggingListener(ctx, logger,
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
	scriptURL := fmt.Sprintf(`%s/invalid.js`, srv.URL)
	err = chromedputil.SourceScript(ctx, scriptURL)
	if err == nil {
		t.Fatal("expected SourceScript to return an error for an invalid script, but it did not")
	}

	// Give the listener a moment to process the events.
	time.Sleep(200 * time.Millisecond)

	// Stop the listener and capture the output.
	t.Logf("Stopping listener")
	cancel()

	<-doneCh

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
