// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"cloudeng.io/file/largefile"
)

// Downloader facilitates downloading files using largefile with
// configurable options.
type Downloader struct {
	readerOptions     []LargeFileOption
	downloaderOptions []largefile.DownloadOption
}

// NewDownloader creates a new Downloader instance.
func NewDownloader() *Downloader {
	return &Downloader{}
}

// WithReaderOptions appends the specified LargeFileOptions to the Downloader.
func (d *Downloader) WithReaderOptions(opts ...LargeFileOption) *Downloader {
	d.readerOptions = append(d.readerOptions, opts...)
	return d
}

// WithDownloaderOptions appends the specified largefile.DownloadOptions to the Downloader.
func (d *Downloader) WithDownloaderOptions(opts ...largefile.DownloadOption) *Downloader {
	d.downloaderOptions = append(d.downloaderOptions, opts...)
	return d
}

// DownloadFile downloads the file from the specified URL to the
// destination path using a temporary file for the download process
// that is renamed to the destination path on successful completion.
// The partial download file has a suffix of ".partialdownload".
func (d *Downloader) DownloadFile(ctx context.Context, u, dest string) (int64, error) {
	tmpName := dest + ".partialdownload"
	if err := os.Remove(tmpName); err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("removing existing partial download %q: %w", tmpName, err)
	}
	defer os.Remove(tmpName) //nolint:errcheck

	_, err := url.Parse(u)
	if err != nil {
		return 0, fmt.Errorf("parsing url %q: %w", u, err)
	}
	rd, err := NewLargeFile(ctx, u, d.readerOptions...)
	if err != nil {
		return 0, fmt.Errorf("creating download reader for %q: %w", u, err)
	}
	wr, err := os.Create(tmpName)
	if err != nil {
		return 0, fmt.Errorf("creating download file %q: %w", dest, err)
	}
	defer wr.Close()
	dl := largefile.NewStreamingDownloader(rd, d.downloaderOptions...)

	errCh := make(chan error, 1)
	var status largefile.StreamingStatus
	go func() {
		var err error
		status, err = dl.Run(ctx)
		errCh <- err
	}()

	n, err := io.Copy(wr, dl.Reader())
	if err != nil {
		return 0, fmt.Errorf("downloading %q to %q: %w", u, dest, err)
	}
	if err := <-errCh; err != nil {
		return 0, fmt.Errorf("downloading %q to %q: %w", u, dest, err)
	}
	if status.DownloadSize != n {
		return 0, fmt.Errorf("downloaded size mismatch for %q: expected %d, got %d", u, status.DownloadSize, n)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return 0, fmt.Errorf("renaming %q to %q: %w", tmpName, dest, err)
	}
	return n, nil
}
