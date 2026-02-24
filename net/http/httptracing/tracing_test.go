// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httptracing_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloudeng.io/net/http/httptracing"
)

func TestTracingHandler(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "hello")
	})

	var reqBodyBuf bytes.Buffer
	bodyCB := func(_ context.Context, _ *slog.Logger, _ *http.Request, data []byte) {
		reqBodyBuf.Write(data)
	}

	th := httptracing.NewTracingHandler(h,
		httptracing.WithTraceHandlerLogger(logger),
		httptracing.WithTraceHandlerRequest(bodyCB))

	reqBody := "request body"
	req := httptest.NewRequest("POST", "/some/path", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	th.ServeHTTP(w, req)

	resp := w.Result()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	body, _ := io.ReadAll(resp.Body)
	if got, want := string(body), "hello"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := reqBodyBuf.String(), reqBody; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	logLines := logBuf.String()
	if strings.Contains(logLines, `"msg":"HTTP Request"`) {
		t.Errorf("unexpected 'HTTP Request' in log: %v", logLines)
	}
	if !strings.Contains(logLines, `"msg":"HTTP Request Completed"`) {
		t.Errorf("missing 'HTTP Request Completed' in log: %v", logLines)
	}
}

func TestTracingRoundTripper(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "hello server")
	}))
	defer server.Close()

	var reqBodyCap, resBodyCap bytes.Buffer

	reqBodyCB := func(_ context.Context, _ *slog.Logger, _ *http.Request, data []byte) {
		reqBodyCap.Write(data)
	}

	resBodyCB := func(_ context.Context, _ *slog.Logger, _ *http.Request, _ *http.Response, data []byte) {
		resBodyCap.Write(data)
	}

	rt := httptracing.NewTracingRoundTripper(
		http.DefaultTransport,
		httptracing.WithTraceLogger(logger),
		httptracing.WithTraceHooks(httptracing.TraceAll),
		httptracing.WithTraceRequest(reqBodyCB),
		httptracing.WithTraceResponse(resBodyCB),
	)

	client := &http.Client{Transport: rt}

	reqBody := "client request body"
	req, err := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Do(req) //nolint:gosec // G704 is overly restrictive here.
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if got, want := res.StatusCode, http.StatusOK; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := reqBodyCap.String(), reqBody; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := resBodyCap.String(), "hello server"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	logs := logBuf.String()
	if strings.Contains(logs, `"msg":"HTTP Request trace"`) {
		t.Errorf("unexpected 'HTTP Request trace' in log: %v", logs)
	}

	// Check for a hook from each category.
	if !strings.Contains(logs, "GetConn") {
		t.Errorf("missing 'GetConn' hook in log: %v", logs)
	}
	if !strings.Contains(logs, "WroteHeaders") {
		t.Errorf("missing 'WroteHeaders' hook in log: %v", logs)
	}
}

func TestTracingRoundTripperHookMask(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "hello")
	}))
	defer server.Close()

	// Test with only DNS hooks enabled.
	rt := httptracing.NewTracingRoundTripper(
		http.DefaultTransport,
		httptracing.WithTraceLogger(logger),
		httptracing.WithTraceHooks(httptracing.TraceDNS),
	)

	client := &http.Client{Transport: rt}
	// Replace 127.0.0.1 with localhost to ensure that a DNS lookup is performed.
	url := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	_, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "DNSStart") {
		t.Errorf("missing 'DNSStart' hook in log: %v", logs)
	}
	if !strings.Contains(logs, "DNSDone") {
		t.Errorf("missing 'DNSDone' hook in log: %v", logs)
	}
	if strings.Contains(logs, "GetConn") {
		t.Errorf("unexpected 'GetConn' hook in log: %v", logs)
	}
	if strings.Contains(logs, "WroteHeaders") {
		t.Errorf("unexpected 'WroteHeaders' hook in log: %v", logs)
	}
}
