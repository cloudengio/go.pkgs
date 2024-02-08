// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs

import (
	"context"
	"fmt"
	"io/fs"
	"sync"

	"cloudeng.io/file"
	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client represents the set of AWS S3 client methods used by s3fs.
type Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetBucketAcl(ctx context.Context, params *s3.GetBucketAclInput, optFns ...func(*s3.Options)) (*s3.GetBucketAclOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
}

func objectHead(ctx context.Context, client Client, bucket, key *string) (*s3.HeadObjectOutput, error) {
	req := s3.HeadObjectInput{
		Bucket: bucket,
		Key:    key,
	}
	return client.HeadObject(ctx, &req)
}

func objectStat(ctx context.Context, client Client, match cloudpath.Match) (file.Info, error) {
	bucket := match.Volume
	awsBucket := aws.String(bucket)
	awsKey := aws.String(match.Key)

	acl, err := bucketAcls.get(ctx, client, bucket)
	if err != nil {
		return file.Info{}, err
	}

	head, err := objectHead(ctx, client, awsBucket, awsKey)
	if err != nil {
		return file.Info{}, err
	}

	var mode fs.FileMode
	if match.Key[len(match.Key)-1] == '/' {
		mode = fs.ModeDir
	}
	var xattr s3xattr
	xattr.owner = aws.ToString(acl.Owner.ID)
	xattr.obj = head

	info := file.NewInfo(
		cloudpath.Base("s3://", byte(match.Separator), match.Key),
		aws.ToInt64(head.ContentLength),
		mode,
		aws.ToTime(head.LastModified),
		xattr,
	)
	return info, nil
}

func getObject(ctx context.Context, client Client, delim byte, name string) (cloudpath.Match, *s3.GetObjectOutput, error) {
	match := cloudpath.AWSS3MatcherSep(name, delim)
	if len(match.Matched) == 0 {
		return match, nil, fmt.Errorf("invalid s3 path: %v", name)
	}
	bucket := match.Volume
	req := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(match.Key),
	}
	res, err := client.GetObject(ctx, &req)
	return match, res, err
}

type bucketACLs struct {
	sync.Mutex
	acls map[string]*s3.GetBucketAclOutput
}

var bucketAcls = &bucketACLs{acls: make(map[string]*s3.GetBucketAclOutput)}

func (bacl *bucketACLs) get(ctx context.Context, client Client, bucket string) (*s3.GetBucketAclOutput, error) {
	bacl.Lock()
	defer bacl.Unlock()
	if acl, ok := bacl.acls[bucket]; ok {
		return acl, nil
	}
	req := s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	}
	res, err := client.GetBucketAcl(ctx, &req)
	if err != nil {
		return nil, err
	}
	bacl.acls[bucket] = res
	return res, nil
}
