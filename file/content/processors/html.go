// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package processor provides support for processing different content types.
package processors

import (
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTML provides support for processing HTML documents.
type HTML struct{}

type HTMLDoc struct {
	root *html.Node
}

func (ho HTML) Parse(rd io.Reader) (HTMLDoc, error) {
	n, err := html.Parse(rd)
	if err != nil {
		return HTMLDoc{n}, err
	}
	return HTMLDoc{root: n}, nil
}

// HREFs returns the hrefs found in the provided HTML document.
func (ho HTMLDoc) HREFs(base string) ([]string, error) {
	return ho.hrefs(base, ho.root), nil
}

func (ho HTMLDoc) resolveReference(base *url.URL, href string) string {
	if idx := strings.LastIndex(href, "/#"); idx != -1 {
		sidx := strings.LastIndex(href, "/")
		if sidx == idx {
			href = href[:idx] + "/index.html" + href[idx+1:]
		}
	}
	if base == nil || len(href) == 0 || href[0] == '#' {
		return href
	}
	vu, err := url.Parse(href)
	if err != nil || vu.IsAbs() {
		return href
	}
	if abs := base.ResolveReference(&url.URL{Path: href}); abs != nil {
		return abs.String()
	}
	return href
}

func (ho HTMLDoc) hrefs(base string, n *html.Node) []string {
	var out []string
	u, _ := url.Parse(base)
	if n.Type == html.ElementNode {
		if n.DataAtom == atom.A || n.DataAtom == atom.Link {
			for _, a := range n.Attr {
				if a.Key == "href" {
					out = append(out, ho.resolveReference(u, a.Val))
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, ho.hrefs(base, c)...)
	}
	return out
}

func (ho HTMLDoc) Title() string {
	return ho.htmlTitleTag(ho.root)
}

func (ho HTMLDoc) htmlTitleTag(n *html.Node) string {
	if n.Type == html.ElementNode {
		if n.DataAtom == atom.Title {
			return n.FirstChild.Data
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := ho.htmlTitleTag(c); len(title) > 0 {
			return title
		}
	}
	return ""
}
