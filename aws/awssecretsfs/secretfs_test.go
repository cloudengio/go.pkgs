// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awssecretsfs_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"testing"

	"cloudeng.io/aws/awssecretsfs"
	"cloudeng.io/aws/awstestutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

var awsInstance *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsInstance)
}

func newSecretsFS() fs.ReadFileFS {
	cfg := awstestutil.DefaultAWSConfig()
	return awssecretsfs.New(cfg,
		awssecretsfs.WithSecretsClient(awsInstance.SecretsManager(cfg)))
}

func TestSecrets(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()

	client := awsInstance.SecretsManager(awstestutil.DefaultAWSConfig())
	s1, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String("test-secret-1"),
		SecretString: aws.String("test-secret-1"),
	})
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	s2, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String("test-secret-2"),
		SecretString: aws.String("test-secret-2"),
	})
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	fs := newSecretsFS()
	for _, tc := range []struct {
		nameOrArn string
		contents  []byte
	}{
		{aws.ToString(s1.Name), []byte("test-secret-1")},
		{aws.ToString(s2.Name), []byte("test-secret-2")},
		{aws.ToString(s1.ARN), []byte("test-secret-1")},
		{aws.ToString(s2.ARN), []byte("test-secret-2")},
	} {
		f, err := fs.Open(tc.nameOrArn)
		if err != nil {
			t.Errorf("%v: %v", tc.nameOrArn, err)
			continue
		}
		data, err := io.ReadAll(f)
		if err != nil {
			t.Errorf("%v: %v", tc.nameOrArn, err)
			continue
		}
		if got, want := data, tc.contents; !bytes.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", tc.nameOrArn, got, want)
		}

		data, err = fs.ReadFile(tc.nameOrArn)
		if err != nil {
			t.Errorf("%v: %v", tc.nameOrArn, err)
			continue
		}
		if got, want := data, tc.contents; !bytes.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", tc.nameOrArn, got, want)
		}

	}
}
