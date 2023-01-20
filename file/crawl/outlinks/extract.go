// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path"
	"strings"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
)

// Download represents a single downloaded file, as opposed to download.Downloaded
// which represents multiple files in the same container. It's a convenience
// for use by the Extractor interface.
type Download struct {
	Request  download.Request
	Download download.Result
}

// Extractor is a lower level interface for outlink extractors that allows
// for the separation of extracting outlinks, filtering/rewriting them and
// creating new download requests to retrieve them. This allows for easier
// customization of the crawl process, for example, to rewrite or otherwise
// manipulate the link names or create appropriate crawl requests for different
// types of outlink.
type Extractor interface {
	// MimeType returns the mime type that this extractor is capable of handling.
	MimeType() string
	// Outlinks extracts outlinks from the specified downloaded file. This
	// is generally specific to the mime type of the content being processed.
	Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error)
	// Request creates new download requests for the specified outlinks.
	Request(depth int, download Download, outlinks []string) download.Request
}

func mimeTypeForPath(p string) string {
	mimeType := mime.TypeByExtension(path.Ext(p))
	if idx := strings.Index(mimeType, ";"); idx > 0 {
		return mimeType[:idx]
	}
	return mimeType
}

// Extract implements crawl.Outlinks.Extract.
func (g *generic) Extract(ctx context.Context, depth int, downloaded download.Downloaded) []download.Request {
	var out []download.Request
	errs := Errors{
		Request: downloaded.Request,
	}
	single := Download{
		Request: downloaded.Request,
	}
	for _, dl := range downloaded.Downloads {
		single.Download = dl
		mimeType := mimeTypeForPath(dl.Name)
		exts, ok := g.extractors[mimeType]
		if !ok {
			errs.Errors = append(errs.Errors, ErrorDetail{
				Result: dl,
				Error:  fmt.Errorf("no extractor for %v", mimeType),
			})
			continue
		}
		for _, ext := range exts {
			links, err := ext.Outlinks(ctx, depth, single, bytes.NewReader(dl.Contents))
			if err != nil {
				errs.Errors = append(errs.Errors, ErrorDetail{
					Result: dl,
					Error:  err,
				})
				continue
			}
			links = g.linkProcessor.Process(links)
			if req := ext.Request(depth, single, links); len(req.Names()) > 0 {
				out = append(out, req)
			}
		}
	}
	if len(errs.Errors) > 0 {
		g.errCh <- errs
	}
	return out
}

type generic struct {
	extractors    map[string][]Extractor
	linkProcessor Process
	errCh         chan<- Errors
}

// NewExtractors creates a crawl.Outlinks.Extractor given instances of
// the lower level Extractor interface. The extractors that match the
// downloaded content's mime type are run for that content.
func NewExtractors(errCh chan<- Errors, processor Process, extractors ...Extractor) crawl.Outlinks {
	ge := &generic{
		extractors:    map[string][]Extractor{},
		linkProcessor: processor,
		errCh:         errCh,
	}
	for _, ext := range extractors {
		ge.extractors[ext.MimeType()] = append(ge.extractors[ext.MimeType()], ext)
	}
	return ge
}
