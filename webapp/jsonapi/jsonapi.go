// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonapi provides utilities for working with json REST APIs.
package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Endpoint represents a JSON API endpoint with a request and response type.
// It provides methods to parse the request from an io.Reader and
// write the response to an io.Writer. If an error occurs during
// parsing or writing, it can write an error response in JSON format using
// the WriteError method.
// It is primarily intended to identify and document JSON API endpoints.
type Endpoint[Req, Resp any] struct{}

// ParseRequest reads the request body from the provided http.Request
// and decodes it into the Request field of the Endpoint.
// If decoding failes, it uses WriteError to write an error message to the
// client.
func (ep Endpoint[Req, Resp]) ParseRequest(rw http.ResponseWriter, r *http.Request, req *Req) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	defer r.Body.Close() // Ensure the body is closed after reading.
	if err := decoder.Decode(req); err != nil {
		WriteErrorMsg(rw, "failed to decode request body", http.StatusBadRequest)
		return fmt.Errorf("failed to decode request body: %w", err)
	}
	_, err := decoder.Token()
	if !errors.Is(err, io.EOF) {
		WriteErrorMsg(rw, "body contains trailing data", http.StatusBadRequest)
		return errors.New("body contains trailing data")
	}
	return nil
}

// WriteResponse writes the response in JSON format to the http.ResponseWriter.
// It sets the Content-Type header to "application/json" and writes the
// HTTP status code. If encoding the response fails, it uses WriteError to write
// an error message to the client.
func (ep Endpoint[Req, Resp]) WriteResponse(rw http.ResponseWriter, resp Resp) error {
	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(resp); err != nil {
		WriteErrorMsg(rw, "failed to encode response", http.StatusInternalServerError)
		return fmt.Errorf("failed to encode response: %w", err)
	}
	rw.WriteHeader(http.StatusOK)
	return nil
}

// ErrorResponse represents a JSON error response.
type ErrorResponse struct {
	Message string `json:"message"`
	// Allow for additional fields in the error response.
}

// WriteErrorMsg writes an error message in JSON format to the http.ResponseWriter
// using WriteErrror.
func WriteErrorMsg(rw http.ResponseWriter, msg string, status int) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(ErrorResponse{Message: msg}) //nolint:errcheck
}

// WriteError writes an ErrorResponse in JSON format to the http.ResponseWriter.
// It sets the appropriate HTTP status code and content type.
func WriteError(rw http.ResponseWriter, err ErrorResponse, status int) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(err) //nolint:errcheck
}
