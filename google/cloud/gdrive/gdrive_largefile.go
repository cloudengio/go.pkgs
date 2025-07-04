// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package gdrive provides an implementation of largefile.Reader for
// Google Drive.
package gdrive

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cloudeng.io/algo/digests"
	"cloudeng.io/file/largefile"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const DefaultLargeFileBlockSize = 1024 * 1024 * 64 // Default block size is 64 MiB.

// DriveReader implements largefile.Reader for Google Drive.
type DriveReader struct {
	service   *drive.Service
	file      *drive.File
	digest    digests.Hash // MD5 checksum of the file, if available.
	blockSize int
}

type options struct {
	blockSize int
}

// Option is used to configure a new DriveReader.
type Option func(*options)

// WithBlockSize sets the preferred block size for downloads.
func WithBlockSize(size int) Option {
	return func(o *options) {
		o.blockSize = size
	}
}

// NewReader creates a new largefile.Reader for a Google Drive file.
// It fetches the file's metadata (name, size, md5 checksum) to initialize the reader.
func NewReader(ctx context.Context, service *drive.Service, fileID string, opts ...Option) (*DriveReader, error) {
	o := options{}
	for _, fn := range opts {
		fn(&o)
	}
	if o.blockSize <= 0 {
		o.blockSize = DefaultLargeFileBlockSize // Default block size is 64 MiB.
	}
	if service == nil {
		return nil, fmt.Errorf("google drive service is nil")
	}

	file, err := service.Files.Get(fileID).Fields("id", "name", "size", "md5Checksum", "sha1Checksum").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get google drive file metadata for %v: %w", fileID, err)
	}
	dr := &DriveReader{
		service:   service,
		file:      file,
		blockSize: o.blockSize,
	}
	var algo string
	var checksum string
	if len(file.Md5Checksum) != 0 {
		algo = digests.MD5
		checksum = file.Md5Checksum
	}
	// prefer SHA1 if available,
	if len(file.Sha1Checksum) != 0 {
		algo = digests.SHA1
		checksum = file.Sha1Checksum
	}
	if len(checksum) != 0 {
		d, err := hex.DecodeString(checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to decode checksum %v %v for file %s: %w", algo, checksum, file.Name, err)
		}
		digest, err := digests.New(algo, d)
		if err != nil {
			return nil, fmt.Errorf("failed to create digest hash for %v %v for file %s: %w", algo, checksum, file.Name, err)
		}
		dr.digest = digest
	}

	return dr, nil
}

// Name implements largefile.Reader.
func (dr *DriveReader) Name() string {
	return dr.file.Name
}

func (dr *DriveReader) FileID() string {
	return dr.file.Id
}

// ContentLengthAndBlockSize implements largefile.Reader.
func (dr *DriveReader) ContentLengthAndBlockSize() (int64, int) {
	return dr.file.Size, dr.blockSize
}

// Digest implements largefile.Reader.
func (dr *DriveReader) Digest() digests.Hash {
	return dr.digest
}

// GetReader implements largefile.Reader.
func (dr *DriveReader) GetReader(ctx context.Context, from, to int64) (io.ReadCloser, largefile.RetryResponse, error) {
	call := dr.service.Files.Get(dr.file.Id)

	rangeHeader := fmt.Sprintf("bytes=%d-%d", from, to)
	call.Header().Set("Range", rangeHeader)

	res, err := call.Context(ctx).Download()
	if err != nil {
		gerr, ok := err.(*googleapi.Error)
		if !ok {
			// Not a google API error, could be a network issue. Treat as retryable.
			return nil, driveRetryResponse{retryable: true}, err
		}
		// Retry on rate-limit/quota errors and server-side errors.
		if gerr.Code == http.StatusForbidden || gerr.Code == http.StatusTooManyRequests || gerr.Code >= 500 {
			return nil, driveRetryResponse{retryable: true}, err
		}
		// Other errors (e.g., 404 Not Found, 401 Unauthorized) are not retryable.
		return nil, driveRetryResponse{retryable: false}, err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		// The DownloadWithContext method may return a response with a non-2xx status
		// code without returning an error.
		defer res.Body.Close()
		return nil, driveRetryResponse{retryable: res.StatusCode >= 500}, fmt.Errorf("google drive download failed with status: %s", res.Status)
	}

	return res.Body, driveRetryResponse{retryable: false}, nil
}

// driveRetryResponse implements largefile.RetryResponse for Google Drive API calls.
type driveRetryResponse struct {
	retryable bool
}

// IsRetryable implements largefile.RetryResponse.
func (r driveRetryResponse) IsRetryable() bool {
	return r.retryable
}

// BackoffDuration implements largefile.RetryResponse. It always returns false,
// indicating that the caller should use a default backoff strategy (e.g., exponential).
func (r driveRetryResponse) BackoffDuration() (bool, time.Duration) {
	return false, 0
}

// GetFileID retrieves a file by its name and returns file metadata including ID and name.
func GetFileID(ctx context.Context, srv *drive.Service, query string) (*drive.File, error) {
	r, err := srv.Files.List().
		Q(query).
		Spaces("drive").
		Fields("files(id, name)").
		PageSize(1).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	if len(r.Files) == 0 {
		return nil, fmt.Errorf("no files found for query %q: %w", query, os.ErrNotExist)
	}
	return r.Files[0], nil
}

// GetWithFields retrieves a file by its ID and returns the file metadata with specified fields.
func GetWithFields(ctx context.Context, srv *drive.Service, fileID string, fields ...googleapi.Field) (*drive.File, error) {
	file, err := srv.Files.Get(fileID).Fields(fields...).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
	}
	if file == nil {
		return nil, fmt.Errorf("%s: %w", fileID, os.ErrNotExist)
	}
	return file, nil
}

func ServiceFromJSON(ctx context.Context, creds []byte, scopes ...string) (*drive.Service, error) {
	// Create a JWT config from the JSON key.
	config, err := google.JWTConfigFromJSON(creds, scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to create JWT: %v", err)
	}

	// Create an HTTP client that is authorized to make requests on behalf of the service account.
	client := config.Client(ctx)

	// Create a new Drive service object using the authenticated HTTP client.
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return srv, nil
}
