// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"

	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (s3fs *T) Put(ctx context.Context, path string, _ fs.FileMode, data []byte) error {
	match := cloudpath.AWSS3MatcherSep(path, s3fs.options.delimiter)
	if len(match.Matched) == 0 {
		return fmt.Errorf("invalid s3 path: %v", path)
	}
	req := s3.PutObjectInput{
		Bucket: aws.String(match.Volume),
		Key:    aws.String(match.Key),
		Body:   bytes.NewReader(data),
	}
	_, err := s3fs.client.PutObject(ctx, &req)
	return err
}

func (s3fs *T) EnsurePrefix(_ context.Context, _ string, _ fs.FileMode) error {
	return nil
}

func (s3fs *T) Get(ctx context.Context, path string) ([]byte, error) {
	_, obj, err := getObject(ctx, s3fs.client, s3fs.options.delimiter, path)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(obj.Body)
}

func (s3fs *T) Delete(ctx context.Context, path string) error {
	match := cloudpath.AWSS3MatcherSep(path, s3fs.options.delimiter)
	if len(match.Matched) == 0 {
		return fmt.Errorf("invalid s3 path: %v", path)
	}
	req := s3.DeleteObjectInput{
		Bucket: aws.String(match.Volume),
		Key:    aws.String(match.Key),
	}
	_, err := s3fs.client.DeleteObject(ctx, &req)
	return err
}

func (s3fs *T) DeleteAll(ctx context.Context, path string) error {
	match := cloudpath.AWSS3MatcherSep(path, s3fs.options.delimiter)
	if len(match.Matched) == 0 {
		return fmt.Errorf("invalid s3 path: %v", path)
	}
	bucket := aws.String(match.Volume)
	prefix := aws.String(match.Key)
	items := aws.Int32(int32(s3fs.options.scanSize))
	var continuationToken *string
	for {
		objs, err := s3fs.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            bucket,
			Prefix:            prefix,
			MaxKeys:           items,
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return err
		}
		if len(objs.Contents) == 0 {
			return nil
		}
		keys := make([]types.ObjectIdentifier, len(objs.Contents))
		for i, obj := range objs.Contents {
			keys[i] = types.ObjectIdentifier{
				Key: obj.Key,
			}
		}
		_, err = s3fs.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: bucket,
			Delete: &types.Delete{Objects: keys},
		})
		if err != nil {
			return err
		}
		if !*objs.IsTruncated {
			return nil
		}
		continuationToken = objs.NextContinuationToken
	}
}
