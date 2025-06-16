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
}

func WithLargeFileBlockSize(blockSize int) LargeFileOption {
	return func(o *largeFileOptions) {
		o.blockSize = blockSize
	}
}

func WithLargeFileLogger(slog *slog.Logger) LargeFileOption {
	return func(o *largeFileOptions) {
		o.logger = slog
	}
}

func WithLargeFileTransport(transport *http.Transport) LargeFileOption {
	return func(o *largeFileOptions) {
		o.transport = transport
	}
}

// OpenLargeFile opens a large file for concurrent reading using file.largefile.Reader.
func NewLargeFile(ctx context.Context, name string, opts ...LargeFileOption) (largefile.Reader, error) {
	lf := &LargeFile{name: name}
	lf.blockSize = 4096 // default block size is 4 KiB
	lf.logger = slog.New(slog.DiscardHandler)
	lf.transport = &http.Transport{}
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
		case http.StatusOK:
			return resp.Body, nil, nil
		case http.StatusServiceUnavailable:
			rr := f.newHTTRetryResponse(resp)
			resp.Body.Close()
			return nil, rr, nil
		default:
			resp.Body.Close()
			return nil, nil, fmt.Errorf("bad status code: %v", resp.Status)
		}
	}
	resp.Body.Close()
	return nil, nil, fmt.Errorf("failed to get reader")
}

func (f *LargeFile) newHTTRetryResponse(response *http.Response) largefile.RetryResponse {
	delay, err := parseRetryAfterHeader(response)
	if err != nil {
		f.logger.Error("failed to parse Retry-After header", "pkg", "cloudeng.io/net/http/httpfs", "retry-after", response.Header.Get("Retry-After"), "error", err)
		return nil // If parsing fails, we cannot retry.
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
	// The standard format for HTTP-dates is RFC1123.
	if retryAt, err := time.Parse(time.RFC1123, retryAfter); err == nil {
		return time.Until(retryAt), nil
	}

	// If neither format works, the header is malformed.
	return 0, fmt.Errorf("unrecognized format for retry-after header: %q", retryAfter)
}
