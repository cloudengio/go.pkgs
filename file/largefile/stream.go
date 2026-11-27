// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile

import (
	"context"
	"fmt"
	"io"
)

// Stream downloads src to w using a StreamingDownloader configured with opts.
// It returns the number of bytes written to w and the status of the download.
// The download is stopped if writing to w fails or ctx is cancelled. If src
// has a digest (see Reader.Digest) and WithDownloadDigest is not used then
// the downloaded data is validated against it, returning ErrDigestMismatch
// if it does not match.
func Stream(ctx context.Context, src Reader, w io.Writer, opts ...DownloadOption) (int64, StreamingStatus, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if d := src.Digest(); d.IsSet() {
		d.Reset()
		opts = append([]DownloadOption{WithDownloadDigest(d)}, opts...)
	}
	dl := NewStreamingDownloader(src, opts...)
	type result struct {
		status StreamingStatus
		err    error
	}
	resCh := make(chan result, 1)
	go func() {
		status, err := dl.Run(ctx)
		resCh <- result{status, err}
	}()

	n, err := io.Copy(w, dl.Reader())
	if err != nil {
		// Unblock Run, which is waiting for the stream to be read.
		cancel()
		dl.pipeRd.CloseWithError(err) //nolint:errcheck
	}
	res := <-resCh
	switch {
	case err != nil:
		return n, res.status, err
	case res.err != nil:
		return n, res.status, res.err
	case n != res.status.DownloadSize:
		return n, res.status, fmt.Errorf("%v: downloaded %d bytes, expected %d", src.Name(), n, res.status.DownloadSize)
	case dl.hash.IsSet() && !dl.hash.Validate():
		return n, res.status, fmt.Errorf("%v: %w", src.Name(), ErrDigestMismatch)
	}
	return n, res.status, nil
}
