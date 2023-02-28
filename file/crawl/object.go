// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"cloudeng.io/file/content"
	"cloudeng.io/file/download"
)

// CrawledObjects returns the downloaded objects as a slice of
// content.Objects using the download.AsObjects function.
func CrawledObjects(crawled Crawled) (objs []content.Object[[]byte, download.Result]) {
	return download.AsObjects(crawled.Downloads)
}
