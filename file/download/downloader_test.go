// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package download_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/download"
	"cloudeng.io/file/filetestutil"
	"cloudeng.io/net/ratecontrol"
)

func runDownloader(ctx context.Context, downloader download.T, reader file.FS, input chan download.Request, output chan download.Downloaded) ([]download.Downloaded, error) {
	nItems := 1000
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errCh <- downloader.Run(ctx, input, output)
		wg.Done()
	}()

	go func() {
		issueDownloadRequests(ctx, nItems, input, reader)
		wg.Done()
	}()

	downloaded := []download.Downloaded{}
	for outs := range output {
		downloaded = append(downloaded, outs)
	}
	err := <-errCh
	wg.Wait()
	return downloaded, err
}

func copyDownloadsToFS(ctx context.Context, t *testing.T, downloaded []download.Downloaded) *filetestutil.WriteFS {
	writeFS := filetestutil.NewWriteFS()
	for _, c := range downloaded {
		for _, d := range c.Downloads {
			if d.Err != nil {
				continue
			}
			f, err := writeFS.Create(ctx, d.Name, fs.FileMode(0600))
			if err != nil {
				t.Fatal(err)
			}
			rd := bytes.NewBuffer(d.Contents)
			if _, err := io.Copy(f, rd); err != nil {
				t.Fatal(err)
			}
			f.Close()
		}
	}
	return writeFS
}

func issueDownloadRequests(ctx context.Context, nItems int, input chan<- download.Request, reader file.FS) {
	for i := 0; i < nItems; i++ {
		select {
		case input <- download.SimpleRequest{RequestedBy: "issueDownloadRequest", FS: reader, Mode: fs.FileMode(0600), Filenames: []string{fmt.Sprintf("%04v", i)}}:
		case <-ctx.Done():
			break
		}
	}
	close(input)
}

func checkForDownloadErrors(t *testing.T, downloaded []download.Downloaded) {
	_, _, line, _ := runtime.Caller(1)
	for _, c := range downloaded {
		for _, d := range c.Downloads {
			if d.Err != nil {
				t.Errorf("line: %v: %v: %v", line, d.Name, d.Err)
			}
		}
	}
}

func TestDownload(t *testing.T) {
	ctx := context.Background()

	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan download.Request, 10)
	output := make(chan download.Downloaded, 10)

	downloader := download.New()

	downloaded, err := runDownloader(ctx, downloader, readFS, input, output)
	if err != nil {
		t.Fatal(err)
	}

	checkForDownloadErrors(t, downloaded)
	writeFS := copyDownloadsToFS(ctx, t, downloaded)
	for _, d := range downloaded {
		if got, want := d.Request.Requester(), "issueDownloadRequest"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	if err := filetestutil.CompareFS(readFS, writeFS); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadCancel(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	src := rand.NewSource(time.Now().UnixMicro())

	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan download.Request, 10)
	output := make(chan download.Downloaded, 10)

	rc := ratecontrol.New(ratecontrol.WithRequestsPerTick(time.Minute, 60))
	downloader := download.New(download.WithRateController(rc, nil))

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, err := runDownloader(ctx, downloader, readFS, input, output)

	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("missing or unexpected error: %v", err)
	}
}

type retryError struct{}

func (e *retryError) Error() string {
	return "retry"
}

func TestDownloadRetries(t *testing.T) {
	ctx := context.Background()

	numRetries := 2
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContentsAfterRetry(src, 8192, numRetries, &retryError{}))
	input := make(chan download.Request, 10)
	output := make(chan download.Downloaded, 10)

	rc := ratecontrol.New(ratecontrol.WithExponentialBackoff(time.Microsecond, 10))
	downloader := download.New(download.WithRateController(rc, &retryError{}))
	downloaded, err := runDownloader(ctx, downloader, readFS, input, output)
	if err != nil {
		t.Fatal(err)
	}

	checkForDownloadErrors(t, downloaded)
	writeFS := copyDownloadsToFS(ctx, t, downloaded)
	if err := filetestutil.CompareFS(readFS, writeFS); err != nil {
		t.Fatal(err)
	}

	for _, d := range downloaded {
		for _, c := range d.Downloads {
			if got, want := c.Retries, numRetries; got != want {
				t.Fatalf("%v: got %v, want %v", c.Name, got, want)
			}
		}
	}
}

func TestDownloadProgress(t *testing.T) {
	ctx := context.Background()

	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan download.Request, 10)
	output := make(chan download.Downloaded, 10)
	errCh := make(chan error, 1)
	progressCh := make(chan download.Progress, 1)
	downloader := download.New(download.WithProgress(time.Millisecond, progressCh, true))

	wg := &sync.WaitGroup{}
	wg.Add(3)

	go func() {
		errCh <- downloader.Run(ctx, input, output)
		wg.Done()
	}()

	nItems := 1000

	go func() {
		issueDownloadRequests(ctx, nItems, input, readFS)
		wg.Done()
	}()

	nDownloads := 0
	go func() {
		for dl := range output {
			nDownloads += len(dl.Downloads)
		}
		wg.Done()
	}()
	lastdownloaded := 0
	nUpdates := 0
	for update := range progressCh {
		lastdownloaded = int(update.Downloaded)
		nUpdates++
	}
	wg.Wait()
	if got, want := nDownloads, nItems; got != want {
		t.Errorf("got %v, want %v\n", got, want)
	}
	if got, want := lastdownloaded, nItems/2; got < want {
		t.Errorf("got %v, want >= %v\n", got, want)
	}
	if got, want := nUpdates, nUpdates/2; got < want {
		t.Errorf("got %v, want >= %v\n", got, want)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
}

func TestDownloadErrors(t *testing.T) {
	ctx := context.Background()

	errFailed := errors.New("failed")
	readFS := filetestutil.NewMockFS(filetestutil.FSErrorOnly(errFailed))
	input := make(chan download.Request, 10)
	output := make(chan download.Downloaded, 10)

	rc := ratecontrol.New(ratecontrol.WithExponentialBackoff(time.Microsecond, 10))
	downloader := download.New(download.WithRateController(rc, &retryError{}))

	downloaded, err := runDownloader(ctx, downloader, readFS, input, output)
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range downloaded {
		for _, c := range d.Downloads {
			if !errors.Is(c.Err, errFailed) {
				t.Fatalf("unexpected error: %v", c.Err)
			}
		}
	}
}
