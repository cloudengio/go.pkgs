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

func (s3fs *s3fs) Put(ctx context.Context, path string, _ fs.FileMode, data []byte) error {
	match := cloudpath.AWSS3Matcher(path)
	if len(match.Matched) == 0 {
		return fmt.Errorf("invalid s3 path: %v", path)
	}
	req := s3.PutObjectInput{
		Bucket: aws.String(match.Volume),
		Key:    aws.String(match.Key),
		Body:   bytes.NewBuffer(data),
	}
	_, err := s3fs.client.PutObject(ctx, &req)
	return err
}

func (s3fs *s3fs) EnsurePrefix(_ context.Context, _ string, _ fs.FileMode) error {
	return nil
}

func (s3fs *s3fs) Get(ctx context.Context, path string) ([]byte, error) {
	_, obj, err := getObject(ctx, s3fs.client, path)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(obj.Body)
}

func (s3fs *s3fs) Delete(ctx context.Context, path string) error {
	match := cloudpath.AWSS3Matcher(path)
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

func (s3fs *s3fs) DeleteAll(ctx context.Context, path string) error {
	match := cloudpath.AWSS3Matcher(path)
	if len(match.Matched) == 0 {
		return fmt.Errorf("invalid s3 path: %v", path)
	}
	bucket, prefix := aws.String(match.Volume), aws.String(match.Key)
	items := aws.Int32(1000)
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
		if *objs.IsTruncated {
			break
		}
		continuationToken = objs.NextContinuationToken
	}
	return nil
}
