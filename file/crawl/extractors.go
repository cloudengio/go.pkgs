// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"context"

	"cloudeng.io/file/download"
	"cloudeng.io/sync/errgroup"
)

type extractorPool struct {
	outlinks           Outlinks
	depth, concurrency int
}

// export this when everything is working.
func newExtractorPool(outlinks Outlinks, depth, concurrency int) *extractorPool {
	return &extractorPool{
		outlinks:    outlinks,
		depth:       depth,
		concurrency: concurrency,
	}
}

func (ep *extractorPool) run(ctx context.Context, downloadedCh <-chan download.Downloaded, crawledCh chan<- Crawled) error {
	// Extractors return when extractorStop is closed.
	var grp errgroup.T
	for i := 0; i < ep.concurrency; i++ {
		i := i
		grp.Go(func() error {
			ep.runExtractor(ctx, i, downloadedCh, crawledCh)
			return nil
		})
	}
	return grp.Wait()
}

func (ep *extractorPool) runExtractor(ctx context.Context,
	id int,
	downloadedCh <-chan download.Downloaded,
	crawledCh chan<- Crawled) {
	for {
		// Wait for newly downloaded items.
		var downloaded download.Downloaded
		var ok bool
		select {
		case <-ctx.Done():
			return
		case downloaded, ok = <-downloadedCh:
			if !ok {
				return
			}
		}
		crawled := Crawled{Depth: ep.depth, Downloaded: downloaded}
		// Extract outlinks.
		extracted := ep.outlinks.Extract(ctx, ep.depth, crawled.Downloaded)
		crawled.Outlinks = append(crawled.Outlinks, extracted...)
		select {
		case <-ctx.Done():
			return
		case crawledCh <- crawled:
		}
	}
}
