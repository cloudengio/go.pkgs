// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httptracing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

type tracingLogger[T any] struct {
	formatter func(v []byte) T
}

func (tl tracingLogger[T]) requestBody(logger *slog.Logger, _ *http.Request, data []byte) {
	if len(data) == 0 {
		logger.Info("HTTP Request Body", "direction", "request", "body", "(empty)")
		return
	}
	logger.Info("HTTP Request Body", "direction", "request", "body",
		tl.formatter(json.RawMessage(data)))
}

func (tl tracingLogger[T]) responseBody(logger *slog.Logger, _ *http.Request, _ *http.Response, data []byte) {
	if len(data) == 0 {
		logger.Info("HTTP Response Body", "direction", "response", "body", "(empty)")
		return
	}
	logger.Info("HTTP Response Body", "direction", "response", "body",
		tl.formatter(json.RawMessage(data)))
}

func (tl tracingLogger[T]) handlerResponseBody(logger *slog.Logger, _ *http.Request, hdr http.Header, statusCode int, data []byte) {
	if len(data) == 0 {
		logger.Info("HTTP Handler Response Body", "direction", "response", "status_code", statusCode, "body", "(empty)")
		return
	}
	logger.Info("HTTP Handler Response Body", "direction", "response", "status_code", statusCode, "body", tl.formatter(json.RawMessage(data)))
}

var jsonFormatter tracingLogger[json.RawMessage] = tracingLogger[json.RawMessage]{
	formatter: func(v []byte) json.RawMessage {
		return json.RawMessage(v)
	},
}

var textFormatter tracingLogger[string] = tracingLogger[string]{
	formatter: func(v []byte) string {
		return string(v)
	},
}

// JSONRequestBodyLogger logs the request body as a JSON object.
// The supplied logger is pre-configured with relevant request information.
func JSONRequestBodyLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte) {
	jsonFormatter.requestBody(logger, nil, data)
}

// JSONResponseBodyLogger logs the response body as a JSON object.
// The supplied logger is pre-configured with relevant request information.
func JSONResponseBodyLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ *http.Response, data []byte) {
	jsonFormatter.responseBody(logger, nil, nil, data)
}

// JSONHandlerResponseLogger logs the response body from an http.Handler as a JSON object.
func JSONHandlerResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ http.Header, statusCode int, data []byte) {
	jsonFormatter.handlerResponseBody(logger, nil, nil, statusCode, data)
}

// TextRequestBodyLogger logs the request body as a text object.
// The supplied logger is pre-configured with relevant request information.
func TextRequestBodyLogger(_ context.Context, logger *slog.Logger, _ *http.Request, data []byte) {
	textFormatter.requestBody(logger, nil, data)
}

// TextResponseBodyLogger logs the response body as a text object.
// The supplied logger is pre-configured with relevant request information.
func TextResponseBodyLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ *http.Response, data []byte) {
	textFormatter.responseBody(logger, nil, nil, data)
}

// TextHandlerResponseLogger logs the response body from an http.Handler as a text object.
func TextHandlerResponseLogger(_ context.Context, logger *slog.Logger, _ *http.Request, _ http.Header, statusCode int, data []byte) {
	textFormatter.handlerResponseBody(logger, nil, nil, statusCode, data)
}
