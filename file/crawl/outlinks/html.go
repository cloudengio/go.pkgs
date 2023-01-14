// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package outlinks

import (
	"context"
	"io"
	"sync"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/download"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTML is an outlink extractor for HTML documents. It implements
// both crawl.Outlinks and outlinks.Extractor.
type HTML struct {
	mu   sync.Mutex
	dups map[string]struct{}
}

func NewHTML() *HTML {
	return &HTML{
		dups: make(map[string]struct{}),
	}
}

func (ho *HTML) MimeType() string {
	return "text/html"
}

// IsDup returns true if link has been seen before (ie. has been used as an
// argument to IsDup).
func (ho *HTML) IsDup(link string) bool {
	ho.mu.Lock()
	defer ho.mu.Unlock()
	if _, ok := ho.dups[link]; ok {
		return true
	}
	ho.dups[link] = struct{}{}
	return false
}

// HREFs returns the hrefs found in the provided HTML document.
func (ho *HTML) HREFs(rd io.Reader) ([]string, error) {
	parsed, err := html.Parse(rd)
	if err != nil {
		return nil, err
	}
	return ho.hrefs(parsed), nil
}

func (ho *HTML) hrefs(n *html.Node) []string {
	var out []string
	if n.Type == html.ElementNode && n.DataAtom == atom.A {
		for _, a := range n.Attr {
			if a.Key == "href" {
				out = append(out, a.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, ho.hrefs(c)...)
	}
	return out
}

// Outlinks implements Extractor.Outlinks.
func (ho *HTML) Outlinks(ctx context.Context, depth int, download Download, contents io.Reader) ([]string, error) {
	if download.Download.Err != nil {
		return nil, nil
	}
	return ho.HREFs(contents)
}

// Request implements Extractor.Request.
func (ho *HTML) Request(depth int, download Download, outlinks []string) download.Request {
	var request crawl.SimpleRequest
	request.Depth = depth
	for _, out := range outlinks {
		if ho.IsDup(out) {
			continue
		}
		request.Filenames = append(request.Filenames, out)
	}
	return request
}
