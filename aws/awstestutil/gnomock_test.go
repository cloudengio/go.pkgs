// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awstestutil_test

import (
	"context"
	"io"
	"slices"
	"testing"

	"cloudeng.io/aws/awstestutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

var awsService *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsService,
		awstestutil.WithS3(),
		awstestutil.WithS3Tree("testdata"),
		awstestutil.WithSecretsManager())
}

func TestSecrets(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	client := awsService.SecretsManager(awstestutil.DefaultAWSConfig())
	list, err := client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(list.SecretList), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	name, secret := "my-secret-name", "my-secret-value"

	create, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(secret),
	})
	if err != nil {
		t.Fatal(err)
	}

	value, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: create.ARN,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := aws.ToString(value.Name), name; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := aws.ToString(value.SecretString), secret; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	list, err = client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(list.SecretList), 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := aws.ToString(list.SecretList[0].Name), name; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func getFile(ctx context.Context, t *testing.T, client *s3.Client, bucket, key string) string {
	_, err := client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		t.Fatal(err)
	}
	obj, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		t.Fatalf("GetObject: %v %v -> %v", bucket, key, err)
	}
	a, err := io.ReadAll(obj.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(a)
}

func TestS3(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	client := awsService.S3(awstestutil.DefaultAWSConfig())

	lb, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(lb.Buckets), 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	buckets := []string{}
	for _, b := range lb.Buckets {
		buckets = append(buckets, aws.ToString(b.Name))
	}
	if got, want := buckets, []string{"bucket-a", "bucket-b"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	a := getFile(ctx, t, client, buckets[0], "f0")
	b := getFile(ctx, t, client, buckets[1], "f0")
	if got, want := a+b, "hello\nworld\n"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
