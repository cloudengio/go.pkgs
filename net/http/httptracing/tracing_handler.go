// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httptracing

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
)

// TracingHandler is an http.Handler that wraps another http.Handler to provide
// basic request tracing. It logs the start and end of each request and can
// be configured to log the request body.
type TracingHandler struct {
	next   http.Handler
	logger *slog.Logger
	opts   handlerOptions
}

// handlerOptions specifies the options for a TracingHandler.
type handlerOptions struct {
	logger       *slog.Logger
	requestBody  TraceRequestBody
	responseBody TraceHandlerResponseBody
}

// TraceHandlerOption is the type for options that can be passed to
// NewTracingHandler.
type TraceHandlerOption func(*handlerOptions)

// WithHandlerLogger provides a logger to be used by the TracingHandler. If not
// specified a default logger that discards all output is used.
func WithHandlerLogger(logger *slog.Logger) TraceHandlerOption {
	return func(o *handlerOptions) {
		o.logger = logger
	}
}

// WithHandlerRequestBody sets a callback to be invoked to log the request body.
// The supplied callback will be called with the request body. The request
// body is read and replaced with a new reader, so the next handler in the
// chain can still read it.
func WithHandlerRequestBody(bl TraceRequestBody) TraceHandlerOption {
	return func(o *handlerOptions) {
		o.requestBody = bl
	}
}

type TraceHandlerResponseBody func(ctx context.Context, logger *slog.Logger, req *http.Request, hdr http.Header, statusCode int, data []byte)

// WithHandlerResponseBody sets a callback to be invoked to log the response body.
// The supplied callback will be called with the response body.
func WithHandlerResponseBody(bl TraceHandlerResponseBody) TraceHandlerOption {
	return func(o *handlerOptions) {
		o.responseBody = bl
	}
}

// NewTracingHandler returns a new TracingHandler that wraps the supplied
// next http.Handler.
func NewTracingHandler(next http.Handler, opts ...TraceHandlerOption) *TracingHandler {
	var o handlerOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.logger == nil {
		o.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return &TracingHandler{
		next:   next,
		logger: o.logger,
		opts:   o,
	}
}

// ServeHTTP implements http.Handler.
func (th *TracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := atomic.AddInt64(&traceID, 1)
	grp := slog.Group("http_handler_trace",
		"traceID", id,
		"method", r.Method,
		"url", r.URL.String(),
		"from", r.RemoteAddr,
	)
	logger := th.logger.With(grp)
	logger.Info("http request started")
	if th.opts.requestBody != nil {
		data, body := copyAndReplace(logger, r.Body)
		r.Body = body
		th.opts.requestBody(r.Context(), logger, r, data)
	}
	trw := &tracingResponseWriter{wr: w}
	th.next.ServeHTTP(trw, r)
	if th.opts.responseBody != nil {
		th.opts.responseBody(r.Context(), logger, r, w.Header(), trw.statusCode, trw.Data())
	} else {
		logger.Info("http request completed", "status", trw.statusCode, "response_size", len(trw.Data()))
	}
}

type tracingResponseWriter struct {
	wr         http.ResponseWriter
	data       []byte
	statusCode int
}

func (trw *tracingResponseWriter) Header() http.Header {
	return trw.wr.Header()
}

func (trw *tracingResponseWriter) Write(data []byte) (int, error) {
	trw.data = append(trw.data, data...)
	return trw.wr.Write(data)
}

func (trw *tracingResponseWriter) WriteHeader(statusCode int) {
	trw.statusCode = statusCode
	trw.wr.WriteHeader(statusCode)
}

func (trw *tracingResponseWriter) Data() []byte {
	return trw.data
}

// Flush implements http.Flusher.
func (trw *tracingResponseWriter) Flush() {
	if f, ok := trw.wr.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker.
func (trw *tracingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := trw.wr.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker not supported by underlying ResponseWriter")
}
