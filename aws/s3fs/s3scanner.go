// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package s3fs

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

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

var slasDelim = aws.String("/")

func NewLevelScanner(client Client, path, delimiter string) filewalk.LevelScanner {
	fmt.Printf("new level scanner: %v\n", path)
	match := cloudpath.AWSS3Matcher(path)
	if len(match.Matched) == 0 {
		return &scanner{err: fmt.Errorf("invalid s3 path: %v", path)}
	}
	key := strings.TrimPrefix(match.Key, "/")
	sc := &scanner{
		client: client,
		match:  match,
		bucket: aws.String(match.Volume),
		prefix: aws.String(key),
		delim:  aws.String(delimiter),
	}
	if IsDirectoryBucket(match.Volume) {
		sc.delim = slasDelim
	}
	return sc
}

func (fs *s3fs) LevelScanner(prefix string) filewalk.LevelScanner {
	return NewLevelScanner(fs.client, prefix, fs.options.delimiter)
}

type scanner struct {
	client            Client
	match             cloudpath.Match
	entries           []filewalk.Entry
	bucket, prefix    *string
	done              bool
	continuationToken *string
	delim             *string
	err               error
}

func (sc *scanner) Contents() []filewalk.Entry {
	return sc.entries
}

func (sc *scanner) Err() error {
	return sc.err
}

func (sc *scanner) Scan(ctx context.Context, n int) bool {
	fmt.Printf("scan: %q %q %v (%v %v)\n", *sc.bucket, *sc.prefix, sc.continuationToken, sc.err, sc.done)
	if sc.err != nil || sc.done {
		return false
	}
	req := s3.ListObjectsV2Input{
		Bucket:            sc.bucket,
		Prefix:            sc.prefix,
		ContinuationToken: sc.continuationToken,
		Delimiter:         sc.delim,
		MaxKeys:           aws.Int32(int32(n)),
	}
	obj, err := sc.client.ListObjectsV2(ctx, &req)
	if err != nil {
		fmt.Printf("ERR %v\n", err)
		sc.err = err
		return false
	}
	fmt.Printf("got.... %v %v\n", len(obj.Contents), len(obj.CommonPrefixes))
	if !*obj.IsTruncated {
		// This is the last response for this directory, save the
		// results and return false on the next call to Scan.
		sc.done = true
	}
	sc.continuationToken = obj.NextContinuationToken
	sc.entries = convertListObjectsOutput(obj, *sc.delim)
	return len(sc.entries) != 0
}

func convertListObjectsOutput(lo *s3.ListObjectsV2Output, delim string) []filewalk.Entry {
	ne := len(lo.Contents) + len(lo.CommonPrefixes)
	if ne == 0 {
		return nil
	}
	entries := make([]filewalk.Entry, ne)
	for i, c := range lo.Contents {
		entries[i].Name = basename(aws.ToString(c.Key), delim)
		entries[i].Type = fs.FileMode(0)
	}
	n := len(lo.Contents)
	for i, p := range lo.CommonPrefixes {
		entries[n+i].Name = aws.ToString(p.Prefix)
		entries[n+i].Type = fs.ModeDir
	}
	return entries
}
