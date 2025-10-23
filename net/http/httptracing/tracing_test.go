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

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	})

	var reqBodyBuf bytes.Buffer
	bodyCB := func(ctx context.Context, l *slog.Logger, req *http.Request, data []byte) {
		reqBodyBuf.Write(data)
	}

	th := httptracing.NewTracingHandler(h,
		httptracing.WithHandlerLogger(logger),
		httptracing.WithHandlerRequestBody(bodyCB))

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
	if !strings.Contains(logLines, "http request started") {
		t.Errorf("missing 'http request started' in log: %v", logLines)
	}
	if !strings.Contains(logLines, "http request completed") {
		t.Errorf("missing 'http request completed' in log: %v", logLines)
	}
}

func TestTracingRoundTripper(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello server")
	}))
	defer server.Close()

	var reqBodyCap, resBodyCap bytes.Buffer

	reqBodyCB := func(ctx context.Context, l *slog.Logger, req *http.Request, data []byte) {
		reqBodyCap.Write(data)
	}

	resBodyCB := func(ctx context.Context, l *slog.Logger, req *http.Request, resp *http.Response, data []byte) {
		resBodyCap.Write(data)
	}

	rt := httptracing.NewTracingRoundTripper(
		http.DefaultTransport,
		httptracing.WithTracingLogger(logger),
		httptracing.WithTraceHooks(httptracing.TraceAll),
		httptracing.WithTraceRequestBody(reqBodyCB),
		httptracing.WithTraceResponseBody(resBodyCB),
	)

	client := &http.Client{Transport: rt}

	reqBody := "client request body"
	req, err := http.NewRequest("POST", server.URL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Do(req)
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
	if !strings.Contains(logs, "HTTP Request trace") {
		t.Errorf("missing 'HTTP Request trace' in log: %v", logs)
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	}))
	defer server.Close()

	// Test with only DNS hooks enabled.
	rt := httptracing.NewTracingRoundTripper(
		http.DefaultTransport,
		httptracing.WithTracingLogger(logger),
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
