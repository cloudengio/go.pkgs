// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloudeng.io/webapp/jsonapi"
)

type TestRequest struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	Address string `json:"address"`
}

type TestResponse struct {
	Greeting string `json:"greeting"`
	ID       int    `json:"id"`
}

func TestEndpoint_ParseRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    bool
		wantStatus int
	}{
		{
			name:       "valid request",
			body:       `{"name": "John", "age": 30, "address": "123 Main St"}`,
			wantErr:    false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       `{"name": "John", "age": 30, "address": "123 Main St"`,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "trailing data",
			body:       `{"name": "John", "age": 30, "address": "123 Main St"} extra`,
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep := jsonapi.Endpoint[TestRequest, TestResponse]{}
			req := httptest.NewRequest(http.MethodPost, "/api", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			var parsedReq TestRequest
			err := ep.ParseRequest(rec, req, &parsedReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if rec.Code != tt.wantStatus {
					t.Errorf("ParseRequest() status = %v, want %v", rec.Code, tt.wantStatus)
				}

				// Verify the error response format
				var errResp jsonapi.ErrorResponse
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if errResp.Message == "" {
					t.Errorf("Expected error message in response, got empty string")
				}
			} else if parsedReq.Name != "John" || parsedReq.Age != 30 || parsedReq.Address != "123 Main St" {
				// Verify the request was correctly parsed
				t.Errorf("Request not correctly parsed: %+v", parsedReq)
			}

		})
	}
}

func TestEndpoint_WriteResponse(t *testing.T) {
	ep := jsonapi.Endpoint[TestRequest, TestResponse]{}
	rec := httptest.NewRecorder()

	resp := TestResponse{
		Greeting: "Hello, world!",
		ID:       42,
	}

	err := ep.WriteResponse(rec, resp)
	if err != nil {
		t.Fatalf("WriteResponse() error = %v", err)
	}

	// Check status code
	if rec.Code != http.StatusOK {
		t.Errorf("WriteResponse() status = %v, want %v", rec.Code, http.StatusOK)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("WriteResponse() Content-Type = %v, want application/json", contentType)
	}

	// Decode and verify response
	var gotResp TestResponse
	if err := json.NewDecoder(rec.Body).Decode(&gotResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if gotResp.Greeting != resp.Greeting || gotResp.ID != resp.ID {
		t.Errorf("Response not correctly encoded: got %+v, want %+v", gotResp, resp)
	}
}

func TestWriteErrorMsg(t *testing.T) {
	rec := httptest.NewRecorder()
	errorMsg := "something went wrong"
	statusCode := http.StatusBadRequest

	jsonapi.WriteErrorMsg(rec, errorMsg, statusCode)

	// Check status code
	if rec.Code != statusCode {
		t.Errorf("WriteErrorMsg() status = %v, want %v", rec.Code, statusCode)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("WriteErrorMsg() Content-Type = %v, want application/json", contentType)
	}

	// Decode and verify error response
	var errResp jsonapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Message != errorMsg {
		t.Errorf("WriteErrorMsg() error = %v, want %v", errResp.Message, errorMsg)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	errorResp := jsonapi.ErrorResponse{
		Message: "custom error message",
	}
	statusCode := http.StatusInternalServerError

	jsonapi.WriteError(rec, errorResp, statusCode)

	// Check status code
	if rec.Code != statusCode {
		t.Errorf("WriteError() status = %v, want %v", rec.Code, statusCode)
	}

	// Check content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("WriteError() Content-Type = %v, want application/json", contentType)
	}

	// Decode and verify error response
	var gotErrResp jsonapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&gotErrResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if gotErrResp.Message != errorResp.Message {
		t.Errorf("WriteError() error = %v, want %v", gotErrResp.Message, errorResp.Message)
	}
}

// TestEndpointEncodeError tests the error handling when encoding the response fails
func TestEndpointEncodeError(t *testing.T) {
	ep := jsonapi.Endpoint[TestRequest, TestResponse]{}

	// Create a custom ResponseWriter that fails on Write
	rec := &failingResponseWriter{
		headers: http.Header{},
	}

	resp := TestResponse{
		Greeting: "Hello, world!",
		ID:       42,
	}

	err := ep.WriteResponse(rec, resp)
	if err == nil {
		t.Fatal("WriteResponse() expected error, got nil")
	}
}

// failingResponseWriter is a mock http.ResponseWriter that fails on Write
type failingResponseWriter struct {
	headers http.Header
	status  int
}

func (f *failingResponseWriter) Header() http.Header {
	return f.headers
}

func (f *failingResponseWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe // Simulate a write error
}

func (f *failingResponseWriter) WriteHeader(statusCode int) {
	f.status = statusCode
}
