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
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

var awsInstance *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsInstance)
}

func newSecretsFS(opts ...awssecretsfs.Option) fs.ReadFileFS {
	cfg := awstestutil.DefaultAWSConfig()
	o := []awssecretsfs.Option{
		awssecretsfs.WithSecretsClient(awsInstance.SecretsManager(cfg))}
	o = append(o, opts...)
	return awssecretsfs.New(cfg, o...)
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

func TestWriteFile(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()

	sfs := newSecretsFS(awssecretsfs.WithAllowCreation(true), awssecretsfs.WithAllowUpdates(true))
	wfs, ok := sfs.(interface {
		WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
	})
	if !ok {
		t.Fatalf("fs.ReadFileFS does not implement WriteFileCtx")
	}

	client := awsInstance.SecretsManager(awstestutil.DefaultAWSConfig())
	secretName := "test-write-secret"
	secretValue := []byte("my-secret-value")

	// 1. Create a new secret.
	err := wfs.WriteFileCtx(ctx, secretName, secretValue, 0600)
	if err != nil {
		t.Fatalf("WriteFileCtx failed to create secret: %v", err)
	}

	// 2. Read it back to verify.
	data, err := sfs.ReadFile(secretName)
	if err != nil {
		t.Fatalf("ReadFile failed after write: %v", err)
	}
	if !bytes.Equal(data, secretValue) {
		t.Errorf("got %q, want %q", data, secretValue)
	}

	// 3. Update an existing secret.
	updatedSecretValue := []byte("my-updated-secret-value")
	err = wfs.WriteFileCtx(ctx, secretName, updatedSecretValue, 0600)
	if err != nil {
		t.Fatalf("WriteFileCtx failed to update secret: %v", err)
	}

	// 4. Read it back to verify the update.
	data, err = sfs.ReadFile(secretName)
	if err != nil {
		t.Fatalf("ReadFile failed after update: %v", err)
	}
	if !bytes.Equal(data, updatedSecretValue) {
		t.Errorf("got %q, want %q", data, updatedSecretValue)
	}

	// 5. Clean up.
	_, err = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(secretName),
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})
	if err != nil {
		t.Errorf("failed to delete secret %v: %v", secretName, err)
	}
}

func TestWriteFilePermissions(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	client := awsInstance.SecretsManager(awstestutil.DefaultAWSConfig())

	wfsFor := func(t *testing.T, opts ...awssecretsfs.Option) interface {
		WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
	} {
		sfs := newSecretsFS(opts...)
		wfs, ok := sfs.(interface {
			WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
		})
		if !ok {
			t.Fatalf("fs.ReadFileFS does not implement WriteFileCtx")
		}
		return wfs
	}

	secretName := "test-write-perms-secret"
	secretValue := []byte("my-secret-value")
	updatedSecretValue := []byte("my-updated-secret-value")

	cleanup := func(name string) {
		_, err := client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:                   aws.String(name),
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
		if err != nil {
			// It may not exist which is fine.
			var rnf *types.ResourceNotFoundException
			if !errors.As(err, &rnf) {
				t.Errorf("failed to delete secret %v: %v", name, err)
			}
		}
	}

	defer cleanup(secretName)

	t.Run("no-create-no-update", func(t *testing.T) {
		wfs := wfsFor(t) // Defaults are false for creation and updates.
		err := wfs.WriteFileCtx(ctx, secretName, secretValue, 0600)
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("expected fs.ErrPermission for create, got %v", err)
		}

		// Manually create the secret.
		_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(secretName),
			SecretString: aws.String(string(secretValue)),
		})
		if err != nil {
			t.Fatalf("failed to create secret: %v", err)
		}
		defer cleanup(secretName)

		err = wfs.WriteFileCtx(ctx, secretName, updatedSecretValue, 0600)
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("expected fs.ErrPermission for update, got %v", err)
		}
	})

	cleanup(secretName)

	t.Run("allow-create-no-update", func(t *testing.T) {
		wfs := wfsFor(t, awssecretsfs.WithAllowCreation(true))
		err := wfs.WriteFileCtx(ctx, secretName, secretValue, 0600)
		if err != nil {
			t.Fatalf("create should have succeeded, but got: %v", err)
		}
		defer cleanup(secretName)

		err = wfs.WriteFileCtx(ctx, secretName, updatedSecretValue, 0600)
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("expected fs.ErrPermission for update, got %v", err)
		}
	})

	cleanup(secretName)

	t.Run("no-create-allow-update", func(t *testing.T) {
		wfs := wfsFor(t, awssecretsfs.WithAllowUpdates(true))
		err := wfs.WriteFileCtx(ctx, secretName, secretValue, 0600)
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("expected fs.ErrPermission for create, got %v", err)
		}

		// Manually create the secret.
		_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(secretName),
			SecretString: aws.String(string(secretValue)),
		})
		if err != nil {
			t.Fatalf("failed to create secret: %v", err)
		}
		defer cleanup(secretName)

		err = wfs.WriteFileCtx(ctx, secretName, updatedSecretValue, 0600)
		if err != nil {
			t.Fatalf("update should have succeeded, but got: %v", err)
		}
	})
}
