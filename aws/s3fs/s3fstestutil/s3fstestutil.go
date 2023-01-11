// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fstestutil

import (
	"context"

	"cloudeng.io/aws/s3fs"
	"cloudeng.io/file"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewMockFS(fs file.FS) s3fs.Client {
	return &mfs{fs: fs}
}

type mfs struct {
	fs file.FS
}

func (m *mfs) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	file, err := m.fs.Open(ctx, *params.Key)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return &s3.GetObjectOutput{
		ContentLength: fi.Size(),
		Body:          file,
	}, nil
}
