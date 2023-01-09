// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package extractors

import (
	"context"
	"fmt"
	"io/fs"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func NewHTML(ch chan<- Errors) crawl.Outlinks {
	return &htmlOutlinks{errCh: ch}
}

type htmlOutlinks struct {
	errCh chan<- Errors
}

func (ho *htmlOutlinks) Extract(ctx context.Context, depth int, downloaded download.Downloaded) []download.Request {
	out := []download.Request{}
	errs := Errors{
		Request:   downloaded.Request,
		Container: downloaded.Container,
	}
	for _, dl := range downloaded.Downloads {
		if dl.Err != nil {
			continue
		}
		o, err := ho.extract(ctx, downloaded.Request, downloaded.Container, depth, dl)
		if err != nil {
			errs.Errors = append(errs.Errors, ErrorDetail{
				Result: dl,
				Error:  err,
			})
			continue
		}
		out = append(out, o...)
	}
	return out
}

func (ho *htmlOutlinks) findlinks(n *html.Node) []string {
	return nil
}

func (ho *htmlOutlinks) extract(ctx context.Context, req download.Request, container fs.FS, depth int, result download.Result) ([]download.Request, error) {
	contents, err := container.Open(result.Name)
	if err != nil {

	}
	doc, err := html.Parse(contents)
	if err != nil {
		return nil, err
	}
	var requests []download.Request
	extractor := func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Href {
			fmt.Printf("XXX %v\n", n.Data)
		}
	}
	extractor(doc)
	return requests, nil
}
