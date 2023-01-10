// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks

import (
	"context"
	"io"
	"io/fs"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
)

// Download represents a single downloaded file, as opposed to download.Downloaded
// which represents multiple files in the same container. It's a convenience
// for use by the Extractor interface.
type Download struct {
	Request   download.Request
	Container fs.FS
	Download  download.Result
}

// Extractor is a lower level interface for outlink extractors that allows
// for the separation of extracting outlinks and creating new download requests
// to retrieve them. This allows for easier customization of the crawl process,
// for example, to rewrite or otherwise manipulate the link names.
type Extractor interface {
	Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error)
	Request(depth int, download Download, outlinks []string) download.Request
}

// Extract implements crawl.Outlinks.Extract.
func (g *generic) Extract(ctx context.Context, depth int, downloaded download.Downloaded) []download.Request {
	var out []download.Request
	errs := Errors{
		Request:   downloaded.Request,
		Container: downloaded.Container,
	}
	single := Download{
		Request:   downloaded.Request,
		Container: downloaded.Container,
	}
	for _, dl := range downloaded.Downloads {
		single.Download = dl
		rd, err := downloaded.Container.Open(dl.Name)
		if err != nil {
			errs.Errors = append(errs.Errors, ErrorDetail{
				Result: dl,
				Error:  err,
			})
			continue
		}
		links, err := g.Outlinks(ctx, depth, single, rd)
		rd.Close()
		if err != nil {
			errs.Errors = append(errs.Errors, ErrorDetail{
				Result: dl,
				Error:  err,
			})
			continue
		}
		if req := g.Request(depth, single, links); len(req.Names()) > 0 {
			out = append(out, req)
		}
	}
	g.errCh <- errs
	return out
}

type generic struct {
	Extractor
	errCh chan<- Errors
}

// NewExtractor creates a crawl.Outlinks.Extractor given an instance of
// the lower level Extractor interface.
func NewExtractor(extractor Extractor, errCh chan<- Errors) crawl.Outlinks {
	return &generic{
		Extractor: extractor,
		errCh:     errCh,
	}
}
