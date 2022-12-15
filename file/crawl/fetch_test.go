package crawl_test

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file/crawl"
	"cloudeng.io/file/filetestutil"
)

type collector struct {
	sync.Mutex
	files map[string][]byte
}

func (c *collector) append(file string, buf []byte) {
	c.Lock()
	defer c.Unlock()
	c.files[file] = append(c.files[file], buf...)
}

type contents struct {
	name      string
	collector *collector
}

func (c *contents) Write(buf []byte) (int, error) {
	c.collector.append(c.name, buf)
	return len(buf), nil
}

func (c *contents) Close() error {
	return nil
}

func (c *collector) Open(name string) (fs.File, error) {
	c.Lock()
	defer c.Unlock()
	contents := c.files[name]
	rdc := &filetestutil.BufferCloser{Buffer: bytes.NewBuffer(contents)}
	fi := filetestutil.NewInfo(name, len(contents), 0600, time.Now(), false, nil)
	return filetestutil.NewFile(rdc, fi), nil
}

func (c *collector) New(name string) (io.WriteCloser, crawl.Request, error) {
	return &contents{collector: c, name: name}, crawl.Request{c, name}, nil
}

func runCrawler(ctx context.Context, crawler crawl.Fetcher, writer crawl.Creator, reader fs.FS, progress chan crawl.Progress, input chan []crawl.Request, output chan []crawl.Downloaded) ([]crawl.Downloaded, error) {
	nItems := 1000
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errCh <- crawler.Run(ctx, writer, progress, input, output)
		wg.Done()
	}()

	go func() {
		crawlItems(ctx, nItems, input, reader)
		wg.Done()
	}()

	crawled := []crawl.Downloaded{}
	for outs := range output {
		crawled = append(crawled, outs...)
	}
	err := <-errCh
	wg.Wait()
	return crawled, err
}

func crawlItems(ctx context.Context, nItems int, input chan<- []crawl.Request, reader fs.FS) {
	for i := 0; i < nItems; i++ {
		select {
		case input <- []crawl.Request{{reader, fmt.Sprintf("%v", i)}}:
		case <-ctx.Done():
			break
		}
	}
	close(input)
}

func sha1Sums(t *testing.T, crawled []crawl.Downloaded) map[string]string {
	_, _, line, _ := runtime.Caller(1)
	s := map[string]string{}
	for _, c := range crawled {
		f, err := c.Container.Open(c.Name)
		if err != nil {
			t.Fatalf("line: %v, %v", line, err)
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("line: %v, %v", line, err)
		}
		sum := sha1.Sum(buf)
		s[c.Name] = hex.EncodeToString(sum[:])
	}
	return s
}
func checkForCrawlErrors(t *testing.T, crawled []crawl.Downloaded) {
	_, _, line, _ := runtime.Caller(1)
	for k, v := range crawled {
		if v.Err != nil {
			t.Errorf("line: %v: %v: %v", line, k, v)
		}
	}
}

func validSHA1Sums(t *testing.T, crawled map[string]string, contents map[string][]byte) {
	_, _, line, _ := runtime.Caller(1)
	if got, want := len(crawled), len(contents); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for cname, csum := range crawled {
		if _, ok := contents[cname]; !ok {
			t.Errorf("line: %v, %v was not crawled", line, cname)
			continue
		}
		sum := sha1.Sum(contents[cname])
		if got, want := csum, hex.EncodeToString(sum[:]); got != want {
			t.Errorf("line: %v, %v: got %v, want %v", line, cname, got, want)
		}
	}
}

func TestCrawl(t *testing.T) {
	ctx := context.Background()

	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan []crawl.Request, 10)
	output := make(chan []crawl.Downloaded, 10)
	writeFS := &collector{files: map[string][]byte{}}

	crawler := crawl.NewFetcher()

	crawled, err := runCrawler(ctx, crawler, writeFS, readFS, nil, input, output)
	if err != nil {
		t.Fatal(err)
	}

	checkForCrawlErrors(t, crawled)
	contents := filetestutil.Contents(readFS)
	validSHA1Sums(t, sha1Sums(t, crawled), contents)
}

func TestCancel(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	src := rand.NewSource(time.Now().UnixMicro())

	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan []crawl.Request, 10)
	output := make(chan []crawl.Downloaded, 10)
	writeFS := &collector{files: map[string][]byte{}}

	crawler := crawl.NewFetcher(crawl.WithRequestsPerMinute(60))

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	_, err := runCrawler(ctx, crawler, writeFS, readFS, nil, input, output)

	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("missing or unexpected error: %v", err)
	}
}

type retryError struct{}

func (e *retryError) Error() string {
	return "retry"
}

func TestRetries(t *testing.T) {
	ctx := context.Background()

	numRetries := 2
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContentsAfterRetry(src, 8192, numRetries, &retryError{}))
	input := make(chan []crawl.Request, 10)
	output := make(chan []crawl.Downloaded, 10)
	writeFS := &collector{files: map[string][]byte{}}

	crawler := crawl.NewFetcher(crawl.WithBackoffParameters(&retryError{},
		time.Microsecond, 10))

	crawled, err := runCrawler(ctx, crawler, writeFS, readFS, nil, input, output)
	if err != nil {
		t.Fatal(err)
	}

	checkForCrawlErrors(t, crawled)
	contents := filetestutil.Contents(readFS)
	validSHA1Sums(t, sha1Sums(t, crawled), contents)

	for _, c := range crawled {
		if got, want := c.Retries, numRetries; got != want {
			t.Fatalf("%v: got %v, want %v", c.Name, got, want)
		}
	}
}

func TestProgress(t *testing.T) {
	ctx := context.Background()

	progressCh := make(chan crawl.Progress, 1)
	src := rand.NewSource(time.Now().UnixMicro())
	readFS := filetestutil.NewMockFS(filetestutil.FSWithRandomContents(src, 8192))
	input := make(chan []crawl.Request, 10)
	output := make(chan []crawl.Downloaded, 10)
	errCh := make(chan error, 1)
	writeFS := &collector{files: map[string][]byte{}}
	crawler := crawl.NewFetcher(crawl.WithProgress(time.Millisecond, progressCh))

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		errCh <- crawler.Run(ctx, writeFS, progressCh, input, output)
		wg.Done()
	}()

	nItems := 1000

	go func() {
		crawlItems(ctx, nItems, input, readFS)
		wg.Done()
	}()

	go func() {
		for range output {
		}
	}()
	lastCrawled := 0
	nUpdates := 0
	for update := range progressCh {
		lastCrawled = int(update.Downloaded)
		nUpdates++
	}
	if got, want := lastCrawled, nItems-100; got < want {
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
