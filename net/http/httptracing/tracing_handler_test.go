// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httptracing

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"cloudeng.io/errors"
)

// mockResponseWriter is a mock implementation of http.ResponseWriter for testing.
type mockResponseWriter struct {
	header      http.Header
	writtenData *bytes.Buffer
	statusCode  int
	flushed     bool
	hijacked    bool
	hijackErr   error
}

func newMockResponseWriter(hijackErr error) *mockResponseWriter {
	return &mockResponseWriter{
		header:      make(http.Header),
		writtenData: &bytes.Buffer{},
		hijackErr:   hijackErr,
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	return m.writtenData.Write(data)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *mockResponseWriter) Flush() {
	m.flushed = true
}

func (m *mockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	m.hijacked = true
	if m.hijackErr != nil {
		return nil, nil, m.hijackErr
	}
	return &net.TCPConn{}, bufio.NewReadWriter(bufio.NewReader(nil), bufio.NewWriter(nil)), nil
}

func TestTracingResponseWriter(t *testing.T) {
	t.Parallel()
	mockRW := newMockResponseWriter(nil)
	trw := &tracingResponseWriter{wr: mockRW}

	// Test Header
	hdr := trw.Header()
	hdr.Set("X-Test", "true")
	if mockRW.Header().Get("X-Test") != "true" {
		t.Error("Header() did not proxy to the underlying ResponseWriter")
	}

	// Test Write
	testData := []byte("hello")
	n, err := trw.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("got %v, want %v", n, len(testData))
	}
	if !bytes.Equal(trw.Data(), testData) {
		t.Errorf("got %s, want %s", trw.Data(), testData)
	}
	if !bytes.Equal(mockRW.writtenData.Bytes(), testData) {
		t.Errorf("got %s, want %s", mockRW.writtenData.Bytes(), testData)
	}

	// Test WriteHeader
	trw.WriteHeader(http.StatusAccepted)
	if trw.statusCode != http.StatusAccepted {
		t.Errorf("got %v, want %v", trw.statusCode, http.StatusAccepted)
	}
	if mockRW.statusCode != http.StatusAccepted {
		t.Errorf("got %v, want %v", mockRW.statusCode, http.StatusAccepted)
	}

	// Test Flush
	trw.Flush()
	if !mockRW.flushed {
		t.Error("Flush() was not called on the underlying ResponseWriter")
	}

	// Test Hijack
	_, _, err = trw.Hijack()
	if err != nil {
		t.Fatalf("Hijack failed: %v", err)
	}
	if !mockRW.hijacked {
		t.Error("Hijack() was not called on the underlying ResponseWriter")
	}
}

func TestTracingResponseWriterImplicit200(t *testing.T) {
	t.Parallel()
	// Write without a prior WriteHeader should record 200, matching net/http behaviour.
	mockRW := newMockResponseWriter(nil)
	trw := &tracingResponseWriter{wr: mockRW}

	if _, err := trw.Write([]byte("hello")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if got, want := trw.statusCode, http.StatusOK; got != want {
		t.Errorf("statusCode after implicit write: got %v, want %v", got, want)
	}
}

func TestTracingResponseWriterNoFlusher(t *testing.T) {
	t.Parallel()
	// A ResponseWriter that doesn't implement http.Flusher.
	mockRW := httptest.NewRecorder()
	trw := &tracingResponseWriter{wr: mockRW}
	trw.Flush() // Should not panic.
}

func TestTracingResponseWriterHijackErrors(t *testing.T) {
	t.Parallel()
	// A ResponseWriter that doesn't implement http.Hijacker.
	mockRW := httptest.NewRecorder()
	trw := &tracingResponseWriter{wr: mockRW}
	_, _, err := trw.Hijack()
	if err == nil || err.Error() != "http.Hijacker not supported by underlying ResponseWriter" {
		t.Errorf("unexpected error: %v", err)
	}

	// A ResponseWriter that returns an error on Hijack.
	hijackErr := errors.New("hijack error")
	mockRW2 := newMockResponseWriter(hijackErr)
	trw2 := &tracingResponseWriter{wr: mockRW2}
	_, _, err = trw2.Hijack()
	if err != hijackErr {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTracingHandlerPanicRecovery(t *testing.T) {
	t.Parallel()

	t.Run("panic-before-write", func(t *testing.T) {
		t.Parallel()
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

		handler := NewTracingHandler(
			http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
				panic("something went wrong")
			}),
			WithTraceHandlerLogger(logger),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req) // must not re-panic

		resp := w.Result()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if got, want := resp.StatusCode, http.StatusInternalServerError; got != want {
			t.Errorf("status: got %v, want %v", got, want)
		}
		if strings.Contains(string(body), "goroutine") {
			t.Errorf("response body contains a stack trace: %s", body)
		}

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "panic in HTTP handler") {
			t.Errorf("log does not contain panic message: %s", logOutput)
		}
		if !strings.Contains(logOutput, "something went wrong") {
			t.Errorf("log does not contain panic value: %s", logOutput)
		}
	})

	t.Run("panic-after-implicit-write", func(t *testing.T) {
		t.Parallel()
		// Handler calls Write (no WriteHeader) then panics.
		// The implicit 200 means trw.statusCode == 200, so http.Error must NOT be
		// attempted (that would corrupt the already-started response).
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

		handler := NewTracingHandler(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("partial"))
				panic("panic after implicit write")
			}),
			WithTraceHandlerLogger(logger),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		// Status must be the implicit 200 — http.Error must not have overwritten it.
		if got, want := resp.StatusCode, http.StatusOK; got != want {
			t.Errorf("status: got %v, want %v", got, want)
		}
		// Body must contain the partial write but no stack trace.
		if !strings.Contains(string(body), "partial") {
			t.Errorf("expected partial body, got: %s", body)
		}
		if strings.Contains(string(body), "goroutine") {
			t.Errorf("response body contains a stack trace: %s", body)
		}
		if !strings.Contains(logBuf.String(), "panic in HTTP handler") {
			t.Errorf("log does not contain panic message: %s", logBuf.String())
		}
	})

	t.Run("panic-after-header", func(t *testing.T) {
		t.Parallel()
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

		handler := NewTracingHandler(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				panic("something went wrong after header")
			}),
			WithTraceHandlerLogger(logger),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req) // must not re-panic

		resp := w.Result()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}

		if strings.Contains(string(body), "goroutine") {
			t.Errorf("response body contains a stack trace: %s", body)
		}
		if !strings.Contains(logBuf.String(), "panic in HTTP handler") {
			t.Errorf("log does not contain panic message: %s", logBuf.String())
		}
	})
}

func TestTracingHandlerResponseBody(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var loggedStatusCode int
	var loggedData []byte

	responseBodyCB := func(_ context.Context, _ *slog.Logger, _ *http.Request, _ http.Header, statusCode int, data []byte) {
		mu.Lock()
		defer mu.Unlock()
		loggedStatusCode = statusCode
		loggedData = append(loggedData, data...)
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "response body")
	})

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	handler := NewTracingHandler(nextHandler,
		WithTraceHandlerLogger(logger),
		WithTraceHandlerResponse(responseBodyCB))

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := string(body), "response body"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	mu.Lock()
	defer mu.Unlock()
	if got, want := loggedStatusCode, http.StatusOK; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := string(loggedData), "response body"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if !strings.Contains(buf.String(), `"msg":"HTTP Request"`) {
		t.Errorf("log output does not contain 'HTTP Request':\n%s", buf.String())
	}
}
