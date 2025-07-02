package largefile_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file/largefile"
)

type syncChain struct {
	before, after chan struct{} // Channel to wait for the next read.

}

// mockStreamingReader implements largefile.Reader for testing.
type mockStreamingReader struct {
	data       []byte
	blockSize  int
	mu         sync.Mutex
	failAt     map[int64]bool      // block offset -> fail
	scheduleAt map[int64]syncChain // Ordered execution of reads.
}

func (m *mockStreamingReader) setOrder(reads ...int) chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scheduleAt == nil {
		m.scheduleAt = make(map[int64]syncChain)
	}
	first := make(chan struct{})
	prev := first
	for _, r := range reads {
		nCh := make(chan struct{})
		m.scheduleAt[int64(r*m.blockSize)] = syncChain{before: prev, after: nCh}
		prev = nCh
	}
	return first
}

func (m *mockStreamingReader) dontWait(from int64) (waiter, closer chan struct{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scheduleAt == nil {
		return nil, nil
	}
	return m.scheduleAt[from].before, m.scheduleAt[from].after
}

func (m *mockStreamingReader) waitFor(from int64) chan struct{} {
	waiter, closer := m.dontWait(from)
	if waiter == nil {
		return make(chan struct{}) // No wait needed, return a closed channel.
	}
	<-waiter      // Wait for the previous read to complete.
	return closer // Signal that this read is done.
}

func (m *mockStreamingReader) Name() string { return "mock" }

func (m *mockStreamingReader) ContentLengthAndBlockSize() (int64, int) {
	return int64(len(m.data)), m.blockSize
}

func (m *mockStreamingReader) Digest() digests.Hash { return digests.Hash{} }

func (m *mockStreamingReader) GetReader(_ context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	m.mu.Lock()
	if m.failAt != nil && m.failAt[from] {
		m.mu.Unlock()
		return nil, nil, errors.New("mock read error")
	}
	if from < 0 || to >= int64(len(m.data)) || from > to {
		return nil, nil, errors.New("invalid range")
	}
	data := m.data[from : to+1]
	m.mu.Unlock()
	doneCh := m.waitFor(from)
	defer close(doneCh)
	return io.NopCloser(bytes.NewReader(data)), nil, nil
}

func TestStreamingDownloader_OrderedDownload(t *testing.T) {
	content := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	blockSize := 8
	reader := &mockStreamingReader{data: content, blockSize: blockSize}

	dl := largefile.NewStreamingDownloader(reader)
	// Run the downloader in a goroutine to simulate streaming.
	go func() {
		_, err := dl.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, dl.Reader())
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), content) {
		t.Errorf("streamed data mismatch: got %q, want %q", buf.Bytes(), content)
	}
}

func TestStreamingDownloader_PartialRead(t *testing.T) {
	content := []byte("abcdefgh")
	blockSize := 2
	reader := &mockStreamingReader{data: content, blockSize: blockSize}

	dl := largefile.NewStreamingDownloader(reader)
	go func() {
		_, err := dl.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	readBuf := make([]byte, 3)
	n, err := io.ReadFull(dl, readBuf) // Must use ReadFull to ensure we read exactly 3 bytes.
	if err != nil && !errors.Is(err, io.EOF) {
		t.Errorf("Read failed: %v", err)
	}
	if n != 3 {
		t.Errorf("Read returned %d bytes, want 3", n)
	}
	if !bytes.Equal(readBuf[:n], []byte("abc")) {
		t.Errorf("Read returned %q, want %q", readBuf[:n], "abc")
	}
}

func TestStreamingDownloader_OutOfOrderBlocks(t *testing.T) {
	content := []byte("abcdefgh")
	blockSize := 2
	reader := &mockStreamingReader{
		data:      content,
		blockSize: blockSize,
	}
	first := reader.setOrder(3, 1, 2, 0) // Ensure blocks are returned out of order.

	var st largefile.StreamingStatus
	errCh := make(chan error)
	dl := largefile.NewStreamingDownloader(reader)
	go func() {
		var err error
		st, err = dl.Run(context.Background())
		errCh <- err
	}()

	go close(first) // Close the first channel to signal the end of the first read.

	got, err := io.ReadAll(dl.Reader())
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("streamed data mismatch: got %q, want %q", got, content)
	}

	if err := <-errCh; err != nil {
		t.Errorf("Run() returned an error: %v", err)
	}
	if st.OutOfOrder == 0 {
		t.Error("expected OutOfOrder > 0, got 0")
	}
}

func TestStreamingDownloader_ErrorPropagation(t *testing.T) {
	content := []byte("abcdefgh")
	blockSize := 2
	reader := &mockStreamingReader{
		data:      content,
		blockSize: blockSize,
		failAt:    map[int64]bool{2: true}, // Fail on block starting at offset 2
	}

	dl := largefile.NewStreamingDownloader(reader)
	errCh := make(chan error, 1)
	go func() {
		_, err := dl.Run(context.Background())
		errCh <- err
	}()

	// Read until we encounter an error
	readBuf := make([]byte, len(content))
	_, err := io.ReadFull(dl.Reader(), readBuf)
	if err == nil {
		t.Error("expected error from ReadFull, got nil")
	}

	// Verify that Run() also returns the error
	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error from Run(), got nil")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for Run() to return error")
	}
}

func TestStreamingDownloader_ContentLength(t *testing.T) {
	content := []byte("abcdefgh")
	blockSize := 2
	reader := &mockStreamingReader{data: content, blockSize: blockSize}
	dl := largefile.NewStreamingDownloader(reader)
	if dl.ContentLength() != int64(len(content)) {
		t.Errorf("ContentLength() = %d, want %d", dl.ContentLength(), len(content))
	}
}

func TestStreamingDownloader_WithDigest(t *testing.T) {
	content := []byte("abcdefgh")
	blockSize := 2
	reader := &mockStreamingReader{data: content, blockSize: blockSize}

	// Calculate expected hash
	h := sha256.New()
	h.Write(content)
	expectedDigest := hex.EncodeToString(h.Sum(nil))

	// Create streaming downloader with digest
	hash, _ := digests.New("sha256", []byte(expectedDigest))
	dl := largefile.NewStreamingDownloader(reader, largefile.WithDownloadDigest(hash))
	go func() {
		_, err := dl.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	// Read all data
	_, err := io.ReadAll(dl.Reader())
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}

	// Verify hash matches expected
	actualDigest := hex.EncodeToString(hash.Sum(nil))
	if actualDigest != expectedDigest {
		t.Errorf("hash mismatch: got %s, want %s", actualDigest, expectedDigest)
	}
}

func TestStreamingDownloader_ContextCancellation(t *testing.T) {
	content := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	blockSize := 8
	reader := &mockStreamingReader{data: content, blockSize: blockSize}

	ctx, cancel := context.WithCancel(context.Background())
	dl := largefile.NewStreamingDownloader(reader)

	errCh := make(chan error, 1)
	go func() {
		_, err := dl.Run(ctx)
		errCh <- err
	}()

	go func() {
		// Need to keep reading from the pipe to simulate a long-running download
		// otherwise the downloader will block writing to the pipe which
		// cannot be interrupted by context cancellation.
		// Claude/Gemini did not understand this.
		for {
			buf := make([]byte, 1)
			_, err := dl.Read(buf)
			if errors.Is(err, io.EOF) {
				break // End of stream
			}
			time.Sleep(10 * time.Millisecond) // Simulate some delay in reading
		}
	}()
	// Cancel the context
	cancel()

	// Verify Run returns with context.Canceled
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for Run() to return")
	}

	// Reading more should fail
	buf := make([]byte, blockSize)
	_, err := dl.Read(buf)
	if err == nil {
		t.Error("expected error after cancellation, got nil")
	}
}

func TestStreamingDownloader_LargeFile(t *testing.T) {
	// Generate a larger content to test multiple blocks
	contentSize := 1024 * 128 // 128 KB
	content := make([]byte, contentSize)
	for i := range content {
		content[i] = byte(i % 256)
	}

	blockSize := 16 * 1024 // 16 KB blocks
	reader := &mockStreamingReader{data: content, blockSize: blockSize}

	dl := largefile.NewStreamingDownloader(reader)
	go func() {
		_, err := dl.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, dl.Reader())
	if err != nil {
		t.Fatalf("io.Copy failed: %v", err)
	}

	if !bytes.Equal(buf.Bytes(), content) {
		t.Error("streamed data mismatch for large file")
	}
}

func testStreamingDownloaderStatus() error {
	content := []byte("abcdefghijklmnop") // 16 bytes
	blockSize := 4                        // 4 blocks
	reader := &mockStreamingReader{
		data:      content,
		blockSize: blockSize,
	}
	first := reader.setOrder(1, 2, 3, 0)

	dl := largefile.NewStreamingDownloader(reader)

	statusCh := make(chan largefile.StreamingStatus, 1)
	errCh := make(chan error, 1)

	go func() {
		status, err := dl.Run(context.Background())
		statusCh <- status
		errCh <- err
	}()

	go close(first) // Close the first channel to signal the end of the first read.

	// We must read from the downloader for it to make progress.
	_, err := io.ReadAll(dl.Reader())
	if err != nil {
		return fmt.Errorf("io.ReadAll failed: %v", err)
	}

	runErr := <-errCh
	if runErr != nil {
		return fmt.Errorf("dl.Run() returned an error: %v", runErr)
	}

	status := <-statusCh

	if status.DownloadedBytes != int64(len(content)) {
		return fmt.Errorf("status.Bytes: got %v, want %v", status.DownloadedBytes, len(content))
	}
	if status.DownloadBlocks != int64(len(content)/blockSize) {
		return fmt.Errorf("status.Blocks: got %v, want %v", status.DownloadBlocks, len(content)/blockSize)
	}
	if status.Duration <= 0 {
		return fmt.Errorf("status.Duration should be > 0, got %v", status.Duration)
	}

	// With the first block delayed, the other 3 should arrive out of order.
	if status.OutOfOrder != 3 {
		return fmt.Errorf("status.OutOfOrder: got %v, want %v", status.OutOfOrder, 3)
	}
	if status.MaxOutOfOrder != 3 {
		return fmt.Errorf("status.MaxOutOfOrder: got %v, want %v", status.MaxOutOfOrder, 3)
	}

	return nil

}

func TestStreamingDownloader_Status(t *testing.T) {
	var err error
	for i := range 2 {
		// this test can fail due to reordering after the reader returns,
		// so we retry to allow for that.
		err := testStreamingDownloaderStatus()
		if err == nil {
			return
		}
		t.Logf("testStreamingDownloaderStatus() failed: attempt %v, err: %v", i, err)
	}
	t.Errorf("testStreamingDownloaderStatus() failed: %v", err)

}

func TestStreamingDownloader_ProgressReporting(t *testing.T) {
	content := make([]byte, 1024)
	blockSize := 128
	reader := &mockStreamingReader{data: content, blockSize: blockSize}

	progressCh := make(chan largefile.DownloadState, 10)
	dl := largefile.NewStreamingDownloader(reader,
		largefile.WithDownloadProgress(progressCh))

	var state largefile.StreamingStatus
	go func() {
		state, _ = dl.Run(context.Background())
		close(progressCh)
	}()

	data, err := io.ReadAll(dl.Reader())
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}

	var lastState largefile.DownloadState
	for state := range progressCh {
		if state.DownloadedBytes < lastState.DownloadedBytes {
			t.Errorf("progress went backwards for bytes: %v -> %v", lastState.DownloadedBytes, state.DownloadedBytes)
		}
		if state.DownloadBlocks < lastState.DownloadBlocks {
			t.Errorf("progress went backwards for blocks: %v -> %v", lastState.DownloadBlocks, state.DownloadBlocks)
		}
		lastState = state
	}

	if len(data) != len(content) {
		t.Errorf("io.ReadAll returned %d bytes, want %d", len(data), len(content))
	}

	if state.DownloadedBytes != int64(len(content)) {
		t.Errorf("final progress bytes: got %v, want %v", state.DownloadedBytes, len(content))
	}
	if state.DownloadBlocks != int64(len(content)/blockSize) {
		t.Errorf("final progress blocks: got %v, want %v", state.DownloadBlocks, len(content)/blockSize)
	}
}
