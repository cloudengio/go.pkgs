// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs_test

import (
	"bytes"
	"context"
	"embed"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"cloudeng.io/aws/s3fs"
	"cloudeng.io/aws/s3fs/s3fstestutil"
	"cloudeng.io/file"
	"github.com/aws/aws-sdk-go-v2/aws"
)

//go:embed testdata
var testdata embed.FS

func TestS3FS(t *testing.T) {
	ctx := context.Background()

	mfs := s3fstestutil.NewMockFS(file.FSFromFS(testdata),
		s3fstestutil.WithBucket("bucket"),
		s3fstestutil.WithLeadingSlashStripped())
	fs := s3fs.New(aws.Config{}, s3fs.WithS3Client(mfs))

	name := "example.html"
	fi, err := fs.Open(ctx, "s3://"+path.Join("bucket", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(fi)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got %s, want %s", got, want)
	}

}
