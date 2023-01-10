// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package crawl

import (
	"cloudeng.io/file/download"
)

// SimpleRequest is a simple implementation of download.Request with
// an additional field to record the depth that the request was created
// at. This will typically be set by an outlink extractor.
type SimpleRequest struct {
	download.SimpleRequest
	Depth int
}
