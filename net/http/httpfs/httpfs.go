// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/net/http/httperror"
)

type Option func(o *options)

type options struct {
	scheme string
}

func WithHTTPScheme() Option {
	return func(o *options) {
		o.scheme = "http"
	}
}

// New creates a new instance of file.FS backed by http/https.
func New(client *http.Client, options ...Option) file.FS {
	fs := &httpfs{client: client}
	fs.options.scheme = "https"
	for _, fn := range options {
		fn(&fs.options)
	}
	return fs
}

type httpfs struct {
	client *http.Client
	options
}

// Scheme implements fs.FS.
func (fs *httpfs) Scheme() string {
	return fs.scheme
}

// Open implements fs.FS.
func (fs *httpfs) Open(name string) (fs.File, error) {
	return fs.OpenCtx(context.Background(), name)
}

// OpenCtx implements file.FS.
func (fs *httpfs) OpenCtx(ctx context.Context, name string) (fs.File, error) {
	req, err := http.NewRequest("GET", name, nil)
	if err != nil {
		return nil, err
	}
	if req.URL.Scheme != fs.scheme {
		return nil, fmt.Errorf("unsupported scheme: %v", req.URL.Scheme)
	}
	req = req.WithContext(ctx)
	resp, err := fs.client.Do(req)
	if err := httperror.CheckResponse(err, resp); err != nil {
		return nil, err
	}
	return &httpFile{ReadCloser: resp.Body, name: name, resp: resp}, nil
}

type httpFile struct {
	io.ReadCloser
	name string
	resp *http.Response
}

// Response is a redacted version of http.Response that can be marshaled
// using gob.
type Response struct {
	// When the response was received.
	When time.Time

	// Fields copied from the http.Response.
	Headers                http.Header
	Trailers               http.Header
	ContentLength          int64
	StatusCode             int
	ProtoMajor, ProtoMinir int
	TransferEncoding       []string
}

func (r *Response) fromHTTPResponse(hr *http.Response) {
	r.Headers = hr.Header
	r.Trailers = hr.Trailer
	r.ContentLength = hr.ContentLength
	r.StatusCode = hr.StatusCode
	r.ProtoMajor = hr.ProtoMajor
	r.ProtoMinir = hr.ProtoMinor
	r.TransferEncoding = hr.TransferEncoding
}

func (f *httpFile) Stat() (fs.FileInfo, error) {
	var lmt time.Time
	if mt := f.resp.Header.Get("Last-Modified"); len(mt) > 0 {
		var err error
		if lmt, err = time.Parse(time.RFC1123, mt); err != nil {
			return nil, err
		}
	}
	resp := &Response{}
	resp.fromHTTPResponse(f.resp)
	fi := file.NewInfo(f.name,
		f.resp.ContentLength,
		0666,
		lmt,
		resp,
	)
	return fi, nil
}
