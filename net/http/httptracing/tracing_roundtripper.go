// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httptracing

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"sync/atomic"
	"time"
)

// TraceHooks is a bitmask to control which httptrace hooks are enabled.
type TraceHooks uint64

const (
	TraceGetConn TraceHooks = 1 << iota
	TraceGotConn
	TracePutIdleConn
	TraceGotFirstResponseByte
	TraceGot100Continue
	TraceGot1xxResponse
	TraceDNSStart
	TraceDNSDone
	TraceConnectStart
	TraceConnectDone
	TraceTLSHandshakeStart
	TraceTLSHandshakeDone
	TraceWroteHeaderField
	TraceWroteHeaders
	TraceWait100Continue
	TraceWroteRequest

	// TraceConnections is a convenience group for connection related hooks.
	TraceConnections = TraceGetConn | TraceGotConn | TracePutIdleConn
	// TraceDNS is a convenience group for DNS hooks.
	TraceDNS = TraceDNSStart | TraceDNSDone
	// TraceConnect is a convenience group for TCP connection hooks.
	TraceConnect = TraceConnectStart | TraceConnectDone
	// TraceTLS is a convenience group for TLS handshake hooks.
	TraceTLS = TraceTLSHandshakeStart | TraceTLSHandshakeDone
	// TraceWrites is a convenience group for request writing hooks.
	TraceWrites = TraceWroteHeaderField | TraceWroteHeaders | TraceWait100Continue | TraceWroteRequest
	// TraceResponses is a convenience group for response related hooks.
	TraceResponses = TraceGotFirstResponseByte | TraceGot100Continue | TraceGot1xxResponse

	// TraceAll enables all available trace hooks.
	TraceAll TraceHooks = TraceConnections | TraceDNS | TraceConnect | TraceTLS | TraceWrites | TraceResponses
)

// TracingRoundTripper is an http.RoundTripper that adds httptrace tracing
// and logging capabilities to an underlying RoundTripper.
type TracingRoundTripper struct {
	next http.RoundTripper
	opts roundtripOptions
}

// TraceRoundtripOption is an option for configuring a TracingRoundTripper.
type TraceRoundtripOption func(*roundtripOptions)

// WithTracingLogger sets the logger to be used for tracing output.
func WithTracingLogger(logger *slog.Logger) TraceRoundtripOption {
	return func(to *roundtripOptions) {
		to.logger = logger
	}
}

// WithTraceHooks sets the trace hooks to be enabled.
func WithTraceHooks(hooks TraceHooks) TraceRoundtripOption {
	return func(to *roundtripOptions) {
		to.traceHooks = hooks
	}
}

// TraceRequestBody is called to log request body data. The supplied data
// is a copy of the original request body.
type TraceRequestBody func(ctx context.Context, logger *slog.Logger, req *http.Request, data []byte)

// TraceResponseBody is called to log response body data. The supplied data
// is a copy of the original response body.
type TraceResponseBody func(ctx context.Context, logger *slog.Logger, req *http.Request, resp *http.Response, data []byte)

// WithTraceRequestBody sets a callback to log request body data.
func WithTraceRequestBody(bl TraceRequestBody) TraceRoundtripOption {
	return func(o *roundtripOptions) {
		o.requestBody = bl
	}
}

// WithTraceResponseBody sets a callback to log response body data.
func WithTraceResponseBody(bl TraceResponseBody) TraceRoundtripOption {
	return func(o *roundtripOptions) {
		o.responseBody = bl
	}
}

type roundtripOptions struct {
	logger       *slog.Logger
	requestBody  TraceRequestBody
	responseBody TraceResponseBody
	traceHooks   TraceHooks
}

// NewTracingRoundTripper creates a new TracingRoundTripper.
func NewTracingRoundTripper(next http.RoundTripper, opts ...TraceRoundtripOption) *TracingRoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	var o roundtripOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.logger == nil {
		o.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return &TracingRoundTripper{next: next, opts: o}
}

func (t *TracingRoundTripper) newClientTrace(logger *slog.Logger) *httptrace.ClientTrace {
	trace := &httptrace.ClientTrace{}
	t.addConnectionTraceHooks(trace, logger, t.opts.traceHooks)
	t.addResponseTraceHooks(trace, logger, t.opts.traceHooks)
	t.addDNSTraceHooks(trace, logger, t.opts.traceHooks)
	t.addConnectTraceHooks(trace, logger, t.opts.traceHooks)
	t.addTLSTraceHooks(trace, logger, t.opts.traceHooks)
	t.addWriteTraceHooks(trace, logger, t.opts.traceHooks)
	return trace
}

func (t *TracingRoundTripper) addConnectionTraceHooks(trace *httptrace.ClientTrace, logger *slog.Logger, hooks TraceHooks) {
	if hooks&TraceGetConn != 0 {
		trace.GetConn = func(hostPort string) {
			logger.Info("GetConn", "hostPort", hostPort)
		}
	}
	if hooks&TraceGotConn != 0 {
		trace.GotConn = func(info httptrace.GotConnInfo) {
			logger.Info("GotConn", "reused", info.Reused, "was_idle", info.WasIdle, "idle_time", info.IdleTime)
		}
	}
	if hooks&TracePutIdleConn != 0 {
		trace.PutIdleConn = func(err error) {
			if err != nil {
				logger.Error("PutIdleConn", "error", err)
				return
			}
			logger.Info("PutIdleConn")
		}
	}
}

func (t *TracingRoundTripper) addResponseTraceHooks(trace *httptrace.ClientTrace, logger *slog.Logger, hooks TraceHooks) {
	if hooks&TraceGotFirstResponseByte != 0 {
		trace.GotFirstResponseByte = func() {
			logger.Info("GotFirstResponseByte")
		}
	}
	if hooks&TraceGot100Continue != 0 {
		trace.Got100Continue = func() {
			logger.Info("Got100Continue")
		}
	}
	if hooks&TraceGot1xxResponse != 0 {
		trace.Got1xxResponse = func(code int, header textproto.MIMEHeader) error {
			logger.Info("Got1xxResponse", "code", code, "header", header)
			return nil
		}
	}
}

func (t *TracingRoundTripper) addDNSTraceHooks(trace *httptrace.ClientTrace, logger *slog.Logger, hooks TraceHooks) {
	if hooks&TraceDNSStart != 0 {
		trace.DNSStart = func(info httptrace.DNSStartInfo) {
			logger.Info("DNSStart", "host", info.Host)
		}
	}
	if hooks&TraceDNSDone != 0 {
		trace.DNSDone = func(info httptrace.DNSDoneInfo) {
			addrs := make([]string, len(info.Addrs))
			for i, addr := range info.Addrs {
				addrs[i] = addr.String()
			}
			logger.Info("DNSDone", "addrs", addrs, "error", info.Err)
		}
	}
}

func (t *TracingRoundTripper) addConnectTraceHooks(trace *httptrace.ClientTrace, logger *slog.Logger, hooks TraceHooks) {
	if hooks&TraceConnectStart != 0 {
		trace.ConnectStart = func(network, addr string) {
			logger.Info("ConnectStart", "network", network, "addr", addr)
		}
	}
	if hooks&TraceConnectDone != 0 {
		trace.ConnectDone = func(network, addr string, err error) {
			if err != nil {
				logger.Error("ConnectDone", "network", network, "addr", addr, "error", err)
				return
			}
			logger.Info("ConnectDone", "network", network, "addr", addr)
		}
	}
}

func (t *TracingRoundTripper) addTLSTraceHooks(trace *httptrace.ClientTrace, logger *slog.Logger, hooks TraceHooks) {
	if hooks&TraceTLSHandshakeStart != 0 {
		trace.TLSHandshakeStart = func() {
			logger.Info("TLSHandshakeStart")
		}
	}
	if hooks&TraceTLSHandshakeDone != 0 {
		trace.TLSHandshakeDone = func(state tls.ConnectionState, err error) {
			if err != nil {
				logger.Error("TLSHandshakeDone", "error", err)
				return
			}
			logger.Info("TLSHandshakeDone", "version", state.Version, "cipher_suite", state.CipherSuite)
		}
	}
}

func (t *TracingRoundTripper) addWriteTraceHooks(trace *httptrace.ClientTrace, logger *slog.Logger, hooks TraceHooks) {
	if hooks&TraceWroteHeaderField != 0 {
		trace.WroteHeaderField = func(key string, value []string) {
			logger.Info("WroteHeaderField", "key", key, "value", value)
		}
	}
	if hooks&TraceWroteHeaders != 0 {
		trace.WroteHeaders = func() {
			logger.Info("WroteHeaders")
		}
	}
	if hooks&TraceWait100Continue != 0 {
		trace.Wait100Continue = func() {
			logger.Info("Wait100Continue")
		}
	}
	if hooks&TraceWroteRequest != 0 {
		trace.WroteRequest = func(info httptrace.WroteRequestInfo) {
			if info.Err != nil {
				logger.Error("WroteRequest", "error", info.Err)
				return
			}
			logger.Info("WroteRequest")
		}
	}
}

var traceID int64

func copyAndReplace(logger *slog.Logger, body io.ReadCloser) ([]byte, io.ReadCloser) {
	if body == nil {
		return nil, nil
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		logger.Error("reading body for logging", "error", err)
		return nil, nil
	}
	// Replace the original reader with a new one containing the same data.
	return data, io.NopCloser(bytes.NewBuffer(data))
}

func (t *TracingRoundTripper) logAndReplaceBody(ctx context.Context, logger *slog.Logger, req *http.Request, resp *http.Response) io.ReadCloser {
	if resp == nil {
		if t.opts.requestBody != nil {
			data, body := copyAndReplace(logger, req.Body)
			t.opts.requestBody(ctx, logger, req, data)
			return body
		}
		logger.Info("HTTP Request trace")
		return req.Body
	}
	if t.opts.responseBody != nil {
		data, body := copyAndReplace(logger, resp.Body)
		t.opts.responseBody(ctx, logger, req, resp, data)
		return body
	}
	return resp.Body
}

// RoundTrip implements the http.RoundTripper interface.
func (t *TracingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	id := atomic.AddInt64(&traceID, 1)

	start := time.Now()

	grp := slog.Group("http_roundtripper_trace",
		"traceID", id,
		"method", req.Method,
		"url", req.URL.String(),
	)
	logger := t.opts.logger.With(grp)

	trace := t.newClientTrace(logger)
	ctx := req.Context()
	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	req.Body = t.logAndReplaceBody(ctx, logger, req, nil)

	// Delegate to the next RoundTripper in the chain
	resp, err := t.next.RoundTrip(req)

	duration := time.Since(start)
	if err != nil {
		logger.Warn("HTTP Response trace",
			"error", err, "duration", duration.String())
	} else {
		logger = logger.With(
			"status", resp.Status,
			"duration", duration.String())
		resp.Body = t.logAndReplaceBody(ctx, logger, req, resp)
	}
	return resp, err
}
