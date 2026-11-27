// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package httpfs

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"cloudeng.io/file/largefile"
)

// Downloader facilitates downloading files using largefile with
// configurable options.
type Downloader struct {
	readerOptions     []LargeFileOption
	downloaderOptions []largefile.DownloadOption
	resume            bool
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

// WithResume enables checkpointed downloads: DownloadFile then uses
// largefile.Download, so that an interrupted download leaves its partial
// state (dest+".partial" and dest+".index", or in the directory set by
// largefile.WithDownloadDir) and a subsequent call for the same file fetches
// only the missing blocks. Without it, any partial download is discarded.
func (d *Downloader) WithResume() *Downloader {
	d.resume = true
	return d
}

// DownloadFile downloads the file from the specified URL to the
// destination path using a temporary file for the download process
// that is renamed to the destination path on successful completion.
// The partial download file has a suffix of ".partialdownload".
// See WithResume for checkpointed downloads.
func (d *Downloader) DownloadFile(ctx context.Context, u, dest string) (int64, error) {
	if d.resume {
		return d.resumeFile(ctx, u, dest)
	}
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
	defer wr.Close() //nolint:errcheck

	n, _, err := largefile.Stream(ctx, rd, wr, d.downloaderOptions...)
	if err != nil {
		return 0, fmt.Errorf("downloading %q to %q: %w", u, dest, err)
	}
	if err := wr.Close(); err != nil {
		// Need to close it for windows before renaming.
		return 0, fmt.Errorf("closing download file %q: %w", dest, err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return 0, fmt.Errorf("renaming %q to %q: %w", tmpName, dest, err)
	}
	return n, nil
}

func (d *Downloader) resumeFile(ctx context.Context, u, dest string) (int64, error) {
	rd, err := NewLargeFile(ctx, u, d.readerOptions...)
	if err != nil {
		return 0, fmt.Errorf("creating download reader for %q: %w", u, err)
	}
	if _, err := largefile.Download(ctx, rd, dest, d.downloaderOptions...); err != nil {
		return 0, fmt.Errorf("downloading %q to %q: %w", u, dest, err)
	}
	size, _ := rd.ContentLengthAndBlockSize()
	return size, nil
}
