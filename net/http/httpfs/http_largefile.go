// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cloudeng.io/file/largefile"
	"cloudeng.io/net/http/httperror"
	"cloudeng.io/net/http/httpfs/rfc9530"
)

// LargeFile implements largefile.Reader for large files accessed via
// HTTP. Such files must support range requests, and the
// "Accept-Ranges" header must be set to "bytes". If the server does not
// support range requests, it returns ErrNoRangeSupport.
// A HEAD request is made to the file to determine its content length
// and digest (if available). The file must be capable of being read
// concurrently in blocks of a specified range. Partial reads are
// treated as errors.
type LargeFile struct {
	largeFileOptions
	name          string
	contentLength int64
	digest        string
}

var ErrNoRangeSupport = &errNoRangeSupport{}

type errNoRangeSupport struct{}

func (e *errNoRangeSupport) Error() string {
	return "does not support range requests"
}

type LargeFileOption func(o *largeFileOptions)

type largeFileOptions struct {
	transport *http.Transport
	logger    *slog.Logger // Optional logger for debugging.
	blockSize int
	delay     time.Duration
}

// WithLargeFileBlockSize sets the block size for reading large files.
func WithLargeFileBlockSize(blockSize int) LargeFileOption {
	return func(o *largeFileOptions) {
		o.blockSize = blockSize
	}
}

// WithLargeFileLogger sets the logger. If not set, a discard logger is used.
func WithLargeFileLogger(slog *slog.Logger) LargeFileOption {
	return func(o *largeFileOptions) {
		o.logger = slog
	}
}

// WithLargeFileTransport sets the HTTP transport for making requests, if not
// set a simple default is used.
func WithLargeFileTransport(transport *http.Transport) LargeFileOption {
	return func(o *largeFileOptions) {
		o.transport = transport
	}
}

// WithDefaultRetryDelay sets the default retry delay for HTTP requests.
// This is used when the server responds with a 503 Service Unavailable status
// but does not provide a Retry-After header or that header cannot be parsed.
// The default value is 1 minute.
func WithDetaultRetryDelay(delay time.Duration) LargeFileOption {
	return func(o *largeFileOptions) {
		o.delay = delay
	}
}

// OpenLargeFile opens a large file for concurrent reading using file.largefile.Reader.
func NewLargeFile(ctx context.Context, name string, opts ...LargeFileOption) (largefile.Reader, error) {
	lf := &LargeFile{name: name}
	lf.blockSize = 4096 // default block size is 4 KiB
	lf.logger = slog.New(slog.DiscardHandler)
	lf.transport = &http.Transport{}
	lf.delay = time.Minute // default retry delay is 1 minute
	for _, opt := range opts {
		opt(&lf.largeFileOptions)
	}
	client := &http.Client{
		Transport: lf.transport,
	}
	req, err := http.NewRequest("HEAD", name, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err := httperror.CheckResponse(err, resp); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return nil, ErrNoRangeSupport
	}
	if digestHeader := resp.Header.Get(rfc9530.ReprDigestHeader); len(digestHeader) > 0 {
		digests, err := rfc9530.ParseReprDigest(digestHeader)
		if err != nil {
			return nil, err
		}
		lf.digest, _ = rfc9530.ChooseDigest(digests, "sha1", "sha256", "sha512")
	}
	lf.contentLength = resp.ContentLength
	return lf, nil
}

// Name implements largefile.Reader.
func (f *LargeFile) Name() string {
	return f.name
}

func (f *LargeFile) ContentLengthAndBlockSize() (int64, int) {
	return f.contentLength, f.blockSize
}

func (f *LargeFile) Digest() string {
	return f.digest
}

func (f *LargeFile) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	client := &http.Client{
		Transport: f.transport,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", f.name, nil)
	req.Header["Range"] = []string{fmt.Sprintf("bytes=%d-%d", from, to)}
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err == nil {
		switch resp.StatusCode {
		case http.StatusOK, http.StatusPartialContent:
			return resp.Body, nil, nil
		case http.StatusServiceUnavailable:
			return nil, f.newHTTRetryResponse(resp), nil
		default:
			bodyPreview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			f.logger.Error("unexpected status code",
				"pkg", "cloudeng.io/net/http/httpfs",
				"url", f.name,
				"status", resp.Status,
				"code", resp.StatusCode,
				"body_preview", string(bodyPreview))
			f.closeBody(resp)
			return nil, nil, fmt.Errorf("bad status code: %v", resp.Status)
		}
	}
	f.closeBody(resp)
	return nil, nil, fmt.Errorf("failed to get reader: %w", err)
}

func (f *LargeFile) closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		// Close the response body to avoid resource leaks.
		resp.Body.Close()
	}
}

func (f *LargeFile) newHTTRetryResponse(resp *http.Response) largefile.RetryResponse {
	defer f.closeBody(resp)
	delay, err := parseRetryAfterHeader(resp)
	if err != nil {
		f.logger.Error("failed to parse Retry-After header",
			"pkg", "cloudeng.io/net/http/httpfs",
			"retry-after", resp.Header.Get("Retry-After"),
			"error", err)
		// Default delay if parsing fails.
		delay = f.delay
	}
	return &httpRetryResponse{
		duration: delay,
	}
}

type httpRetryResponse struct {
	duration time.Duration
}

func (r *httpRetryResponse) IsRetryable() bool {
	return true
}

func (r *httpRetryResponse) BackoffDuration() (bool, time.Duration) {
	return r.duration != 0, r.duration
}

// ParseRetryAfter parses the value of a Retry-After header.
// It returns a struct containing the calculated retry time and the delay duration,
// or an error if the header is missing or malformed.
func parseRetryAfterHeader(response *http.Response) (time.Duration, error) {
	if response == nil {
		return 0, nil
	}
	retryAfter := response.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0, nil // No Retry-After header, no delay.
	}

	if retryAfter == "" {
		return 0, nil
	}

	// First, try to parse it as an integer (delay-seconds).
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		if seconds < 0 {
			return 0, fmt.Errorf("invalid negative value for retry-after in seconds: %d", seconds)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	// If it's not an integer, try to parse it as an HTTP-date.
	if retryAt, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(retryAt), nil
	}

	// If neither format works, the header is malformed.
	return 0, fmt.Errorf("unrecognized format for retry-after header: %q", retryAfter)
}
