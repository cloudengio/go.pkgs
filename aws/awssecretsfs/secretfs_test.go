// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awssecretsfs_test

import (
	"bytes"
	"context"
	"errors"
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

	sfs := newSecretsFS()
	for _, tc := range []struct {
		nameOrArn string
		contents  []byte
	}{
		{aws.ToString(s1.Name), []byte("test-secret-1")},
		{aws.ToString(s2.Name), []byte("test-secret-2")},
		{aws.ToString(s1.ARN), []byte("test-secret-1")},
		{aws.ToString(s2.ARN), []byte("test-secret-2")},
	} {
		f, err := sfs.Open(tc.nameOrArn)
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

		data, err = sfs.ReadFile(tc.nameOrArn)
		if err != nil {
			t.Errorf("%v: %v", tc.nameOrArn, err)
			continue
		}
		if got, want := data, tc.contents; !bytes.Equal(got, want) {
			t.Errorf("%v: got %v, want %v", tc.nameOrArn, got, want)
		}
	}

	_, err = sfs.Open("non-existent-secret")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %v", err)
	}

	_, err = sfs.ReadFile("non-existent-secret")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %v", err)
	}

}

func TestSecretDeletion(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()

	client := awsInstance.SecretsManager(awstestutil.DefaultAWSConfig())
	s, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String("test-secret-to-delete"),
		SecretString: aws.String("to-be-deleted"),
	})
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}

	sfs := newSecretsFS()
	name := aws.ToString(s.Name)

	// Check it exists first.
	data, err := sfs.ReadFile(name)
	if err != nil {
		t.Fatalf("%v: %v", name, err)
	}
	if got, want := string(data), "to-be-deleted"; got != want {
		t.Fatalf("got %v, want %v", got, want)
	}

	// Now delete it.
	_, err = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   s.ARN,
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	if err != nil {
		t.Fatalf("failed to delete secret: %v", err)
	}

	// Now it should fail.
	_, err = sfs.ReadFile(name)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected fs.ErrNotExist, got %v", err)
	}
}
