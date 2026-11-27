// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package largefile_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file/largefile"
)

// digestReader is a sliceReader with a digest.
type digestReader struct {
	*sliceReader
	digest digests.Hash
}

func (d digestReader) Digest() digests.Hash { return d.digest }

func newDigest(t *testing.T, data []byte) digests.Hash {
	t.Helper()
	sum := sha256.Sum256(data)
	h, err := digests.New(digests.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func exists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func TestDownload(t *testing.T) {
	data := testData(100*40 + 17)
	dest := filepath.Join(t.TempDir(), "file")
	src := digestReader{&sliceReader{data: data, blockSize: 100}, newDigest(t, data)}
	st, err := largefile.Download(t.Context(), src, dest, largefile.WithDownloadConcurrency(4))
	if err != nil || !st.Complete {
		t.Fatalf("%+v: %v", st, err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, data) {
		t.Errorf("got %v bytes, %v", len(got), err)
	}
	if exists(dest+".partial") || exists(dest+".index") {
		t.Error("partial files were not removed")
	}
}

// failAfter fails every block from the given offset on, once.
type failFrom struct {
	*sliceReader
	from int64
}

func (f failFrom) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	if from >= f.from {
		return nil, nil, io.ErrUnexpectedEOF
	}
	return f.sliceReader.GetReader(ctx, from, to)
}

func TestDownloadResume(t *testing.T) {
	const blockSize, nblocks = 100, 40
	data := testData(blockSize*nblocks + 17)
	dest := filepath.Join(t.TempDir(), "file")

	// The first attempt fails part way through.
	r1 := failFrom{&sliceReader{data: data, blockSize: blockSize}, 20 * blockSize}
	st, err := largefile.Download(t.Context(), r1, dest, largefile.WithDownloadConcurrency(1))
	if err == nil || st.Complete {
		t.Fatalf("expected a failed download: %+v: %v", st, err)
	}
	if !st.Resumable || !exists(dest+".partial") || !exists(dest+".index") || exists(dest) {
		t.Fatalf("expected a resumable download: %+v", st)
	}

	// The second only fetches the blocks that remain.
	r2 := &sliceReader{data: data, blockSize: blockSize}
	st, err = largefile.Download(t.Context(), r2, dest, largefile.WithDownloadConcurrency(1))
	if err != nil || !st.Complete {
		t.Fatalf("%+v: %v", st, err)
	}
	if got, want := r2.opened.Load(), int64(nblocks+1-20); got != want {
		t.Errorf("fetched %v blocks, want %v", got, want)
	}
	if got, err := os.ReadFile(dest); err != nil || !bytes.Equal(got, data) {
		t.Errorf("got %v bytes, %v", len(got), err)
	}
}

func TestDownloadStalePartial(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "file")
	old := testData(100 * 10)
	r1 := failFrom{&sliceReader{data: old, blockSize: 100}, 500}
	if _, err := largefile.Download(t.Context(), r1, dest); err == nil {
		t.Fatal("expected an error")
	}
	// A file of a different size must not reuse the partial download.
	data := testData(100*12 + 1)
	if _, err := largefile.Download(t.Context(), &sliceReader{data: data, blockSize: 100}, dest); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(dest); err != nil || !bytes.Equal(got, data) {
		t.Errorf("got %v bytes, %v", len(got), err)
	}
}

func TestDownloadDigestMismatch(t *testing.T) {
	data := testData(100 * 10)
	wrong := newDigest(t, []byte("something else"))
	dest := filepath.Join(t.TempDir(), "file")
	src := digestReader{&sliceReader{data: data, blockSize: 100}, wrong}
	st, err := largefile.Download(t.Context(), src, dest)
	if !errors.Is(err, largefile.ErrDigestMismatch) || st.Resumable {
		t.Errorf("%+v: %v", st, err)
	}
	if exists(dest) || exists(dest+".partial") || exists(dest+".index") {
		t.Error("files should not remain")
	}

	var out bytes.Buffer
	src = digestReader{&sliceReader{data: data, blockSize: 100}, wrong}
	if _, _, err := largefile.Stream(t.Context(), src, &out); !errors.Is(err, largefile.ErrDigestMismatch) {
		t.Errorf("Stream: got %v", err)
	}
	src = digestReader{&sliceReader{data: data, blockSize: 100}, newDigest(t, data)}
	out.Reset()
	if _, _, err := largefile.Stream(t.Context(), src, &out); err != nil {
		t.Errorf("Stream: %v", err)
	}
}
