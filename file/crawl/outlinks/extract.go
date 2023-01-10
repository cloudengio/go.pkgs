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

type Download struct {
	Request   download.Request
	Container fs.FS
	Download  download.Result
}

type Extractor interface {
	Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error)
	Request(depth int, download Download, outlinks []string) download.Request
}

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

func NewExtractor(extractor Extractor, errCh chan<- Errors) crawl.Outlinks {
	return &generic{
		Extractor: extractor,
		errCh:     errCh,
	}
}
