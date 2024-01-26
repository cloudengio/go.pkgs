// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"

	"cloudeng.io/file/filewalk"
	"cloudeng.io/path/cloudpath"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	directoryBucketRE   = regexp.MustCompile(`--.*--x-s3$`)
	directoryBucketAZRE = regexp.MustCompile(`--(.*)--x-s3$`)
)

func IsDirectoryBucket(bucket string) bool {
	return directoryBucketRE.MatchString(bucket)
}

func DirectoryBucketAZ(bucket string) string {
	m := directoryBucketAZRE.FindStringSubmatch(bucket)
	if len(m) == 0 {
		return ""
	}
	return m[1]
}

var initialStartAfter = aws.String("")
var slashDelimiter = aws.String("/")

func NewLevelScanner(client Client, path string) filewalk.LevelScanner {
	match := cloudpath.AWSS3Matcher(path)
	if len(match.Matched) == 0 {
		return &scannerCommon{err: fmt.Errorf("invalid s3 path: %v", path)}
	}
	sc := scannerCommon{
		client:    client,
		match:     match,
		bucket:    aws.String(match.Volume),
		prefix:    aws.String(match.Key),
		delimiter: slashDelimiter,
	}
	if IsDirectoryBucket(match.Volume) {
		return &dbScanner{scannerCommon: sc,
			continuationToken: nil,
		}
	}
	return &dbScanner{scannerCommon: sc,
		continuationToken: nil,
	}
	return &gpScanner{scannerCommon: sc,
		startAfter: initialStartAfter,
	}
}

func (fs *s3fs) LevelScanner(prefix string) filewalk.LevelScanner {
	return NewLevelScanner(fs.client, prefix)
}

type scannerCommon struct {
	client         Client
	match          cloudpath.Match
	entries        []filewalk.Entry
	bucket, prefix *string
	delimiter      *string
	err            error
}

func (s *scannerCommon) Contents() []filewalk.Entry {
	return s.entries
}

func (s *scannerCommon) Err() error {
	return s.err
}

func (s *scannerCommon) Scan(ctx context.Context, n int) bool {
	return false
}

type gpScanner struct {
	scannerCommon
	startAfter *string
}

func (s *gpScanner) Scan(ctx context.Context, n int) bool {
	if s.err != nil {
		return false
	}
	req := s3.ListObjectsV2Input{
		Bucket:     s.bucket,
		Prefix:     s.prefix,
		StartAfter: s.startAfter,
		Delimiter:  s.delimiter,
		MaxKeys:    aws.Int32(int32(n)),
	}
	obj, err := s.client.ListObjectsV2(ctx, &req)
	if err != nil {
		s.err = err
		return false
	}
	kc := aws.ToInt32(obj.KeyCount)
	if kc == 0 {
		return false
	}
	s.entries = convertListObjectsOutput(obj)
	last := s.entries[len(s.entries)-1].Name
	s.startAfter = aws.String(last)
	return true
}

type dbScanner struct {
	scannerCommon
	continuationToken *string
	done              bool
}

func (s *dbScanner) Scan(ctx context.Context, n int) bool {
	if s.err != nil || s.done {
		return false
	}
	req := s3.ListObjectsV2Input{
		Bucket:            s.bucket,
		Prefix:            s.prefix,
		ContinuationToken: s.continuationToken,
		Delimiter:         slashDelimiter,
		MaxKeys:           aws.Int32(int32(n)),
	}
	obj, err := s.client.ListObjectsV2(ctx, &req)
	if err != nil {
		s.err = err
		return false
	}
	if !*obj.IsTruncated {
		// This is the last response for this directory, save the
		// results and return false on the next call to Scan.
		s.done = true
	}
	s.continuationToken = obj.NextContinuationToken
	s.entries = convertListObjectsOutput(obj)
	return len(s.entries) != 0
}

func convertListObjectsOutput(lo *s3.ListObjectsV2Output) []filewalk.Entry {
	ne := len(lo.Contents) + len(lo.CommonPrefixes)
	if ne == 0 {
		return nil
	}
	entries := make([]filewalk.Entry, ne)
	for i, c := range lo.Contents {
		entries[i].Name = aws.ToString(c.Key)
		entries[i].Type = fs.FileMode(0)
	}
	n := len(lo.Contents)
	for i, p := range lo.CommonPrefixes {
		entries[n+i].Name = aws.ToString(p.Prefix)
		entries[n+i].Type = fs.ModeDir
	}
	return entries
}
