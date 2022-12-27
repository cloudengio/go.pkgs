package download_test

import (
	"context"
	"errors"
	"fmt"
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
)

type dlRequest struct {
	container fs.FS
	names     []string
}

func (dlr dlRequest) Container() fs.FS {
	return dlr.container
}

func (dlr dlRequest) Names() []string {
	return dlr.names
}

func (dlr dlRequest) FileMode() fs.FileMode {
	return fs.FileMode(0600)
}

func runDownloader(ctx context.Context, downloader download.T, writer file.WriteFS, reader fs.FS, input chan download.Request, output chan download.Downloaded) ([]download.Downloaded, error) {
	nItems := 1000
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errCh <- downloader.Run(ctx, writer, input, output)
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

func issueDownloadRequests(ctx context.Context, nItems int, input chan<- download.Request, reader fs.FS) {
	for i := 0; i < nItems; i++ {
		select {
		case input <- dlRequest{container: reader, names: []string{fmt.Sprintf("%v", i)}}:
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
	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)

	downloader := download.New()

	downloaded, err := runDownloader(ctx, downloader, writeFS, readFS, input, output)
	if err != nil {
		t.Fatal(err)
	}

	checkForDownloadErrors(t, downloaded)
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
	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)

	downloader := download.New(download.WithRequestsPerMinute(60))

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, err := runDownloader(ctx, downloader, writeFS, readFS, input, output)

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
	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)

	downloader := download.New(
		download.WithBackoffParameters(&retryError{}, time.Microsecond, 10))

	downloaded, err := runDownloader(ctx, downloader, writeFS, readFS, input, output)
	if err != nil {
		t.Fatal(err)
	}

	checkForDownloadErrors(t, downloaded)
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
	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)
	progressCh := make(chan download.Progress, 1)
	downloader := download.New(download.WithProgress(time.Millisecond, progressCh, true))

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errCh <- downloader.Run(ctx, writeFS, input, output)
		wg.Done()
	}()

	nItems := 1000

	go func() {
		issueDownloadRequests(ctx, nItems, input, readFS)
		wg.Done()
	}()

	go func() {
		for range output {
		}
	}()
	lastdownloaded := 0
	nUpdates := 0
	for update := range progressCh {
		lastdownloaded = int(update.Downloaded)
		nUpdates++
	}
	if got, want := lastdownloaded, nItems-100; got < want {
		t.Errorf("got %v, want >= %v\n", got, want)
	}
	if got, want := nUpdates, 10; got < want {
		t.Errorf("got %v, want >= %v\n", got, want)
	}
	wg.Wait()
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
	writeFS := filetestutil.NewMockFS(filetestutil.FSWriteFS()).(file.WriteFS)

	downloader := download.New(download.WithBackoffParameters(&retryError{},
		time.Microsecond, 10))

	downloaded, err := runDownloader(ctx, downloader, writeFS, readFS, input, output)
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
