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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloudeng.io/aws/awssecretsfs"
	"cloudeng.io/aws/awstestutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// mockClient implements awssecretsfs.Client for unit tests that do not require
// a real AWS connection. Only GetSecretValue is wired; all other methods panic
// because the singleflight tests use ARN-format keys (bypassing DescribeSecret).
type mockClient struct {
	calls          atomic.Int64
	getSecretValue func(ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func (m *mockClient) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	m.calls.Add(1)
	return m.getSecretValue(ctx, input, optFns...)
}

func (m *mockClient) ListSecretVersionIds(_ context.Context, _ *secretsmanager.ListSecretVersionIdsInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretVersionIdsOutput, error) { //nolint:revive
	panic("not implemented")
}

func (m *mockClient) DeleteSecret(_ context.Context, _ *secretsmanager.DeleteSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	panic("not implemented")
}

func (m *mockClient) PutSecretValue(_ context.Context, _ *secretsmanager.PutSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
	panic("not implemented")
}

func (m *mockClient) CreateSecret(_ context.Context, _ *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	panic("not implemented")
}

func (m *mockClient) DescribeSecret(_ context.Context, _ *secretsmanager.DescribeSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
	panic("not implemented")
}

var awsInstance *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsInstance)
}

func newSecretsFS(opts ...awssecretsfs.Option) *awssecretsfs.T {
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
	err = sfs.Delete(ctx, name)
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

	client := awsInstance.SecretsManager(awstestutil.DefaultAWSConfig())
	secretName := "test-write-secret" //nolint:gosec // G101 this is not a hardcoded secret.
	secretValue := []byte("my-secret-value")

	// 1. Create a new secret.
	err := sfs.WriteFileCtx(ctx, secretName, secretValue, 0600)
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
	err = sfs.WriteFileCtx(ctx, secretName, updatedSecretValue, 0600)
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

	wfsFor := func(opts ...awssecretsfs.Option) interface {
		WriteFileCtx(ctx context.Context, name string, data []byte, perm fs.FileMode) error
	} {
		return newSecretsFS(opts...)
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
		wfs := wfsFor() // Defaults are false for creation and updates.
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
		wfs := wfsFor(awssecretsfs.WithAllowCreation(true))
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
		wfs := wfsFor(awssecretsfs.WithAllowUpdates(true))
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

// TestSingleFlightDeduplicates verifies that concurrent ReadFileCtx calls for the
// same key are coalesced into a single backend GetSecretValue invocation, and that
// calls for different keys are not coalesced.
//
// ARN-format keys are used so readSecret bypasses DescribeSecret entirely, keeping
// the mock simple.
func TestSingleFlightDeduplicates(t *testing.T) {
	const (
		n         = 10
		secretARN = "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret" //nolint:gosec //G101: Potential hardcoded credentials false positive
		wantData  = "secret-value"
	)

	t.Run("SameKeyCoalesced", func(t *testing.T) {
		started := make(chan struct{})
		gate := make(chan struct{})
		var signalOnce sync.Once

		mock := &mockClient{
			getSecretValue: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				signalOnce.Do(func() { close(started) })
				<-gate
				return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(wantData)}, nil
			},
		}
		sfs := awssecretsfs.NewSecretsFS(aws.Config{}, awssecretsfs.WithSecretsClient(mock))

		results := make([][]byte, n)
		errs := make([]error, n)
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				results[i], errs[i] = sfs.ReadFileCtx(context.Background(), secretARN)
			}(i)
		}

		// Wait for the first GetSecretValue call to enter, then give the other
		// goroutines a moment to queue up inside the singleflight group.
		<-started
		time.Sleep(20 * time.Millisecond)
		close(gate)
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", i, err)
			}
			if !bytes.Equal(results[i], []byte(wantData)) {
				t.Errorf("goroutine %d: got %q, want %q", i, results[i], wantData)
			}
		}
		if got := mock.calls.Load(); got != 1 {
			t.Errorf("GetSecretValue call count: got %d, want 1", got)
		}
	})

	t.Run("DifferentKeysNotCoalesced", func(t *testing.T) {
		const (
			arnA = "arn:aws:secretsmanager:us-east-1:123456789012:secret:key-a"
			arnB = "arn:aws:secretsmanager:us-east-1:123456789012:secret:key-b"
		)

		started := make(chan struct{})
		gate := make(chan struct{})
		var signalOnce sync.Once

		mock := &mockClient{
			getSecretValue: func(_ context.Context, input *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				signalOnce.Do(func() { close(started) })
				<-gate
				return &secretsmanager.GetSecretValueOutput{SecretString: aws.String(aws.ToString(input.SecretId))}, nil
			},
		}
		sfs := awssecretsfs.NewSecretsFS(aws.Config{}, awssecretsfs.WithSecretsClient(mock))

		var wg sync.WaitGroup
		for range n {
			wg.Add(2)
			go func() { defer wg.Done(); sfs.ReadFileCtx(context.Background(), arnA) }() //nolint:errcheck
			go func() { defer wg.Done(); sfs.ReadFileCtx(context.Background(), arnB) }() //nolint:errcheck
		}

		<-started
		time.Sleep(20 * time.Millisecond)
		close(gate)
		wg.Wait()

		// Two distinct keys → exactly two backend calls, one per key.
		if got := mock.calls.Load(); got != 2 {
			t.Errorf("GetSecretValue call count: got %d, want 2", got)
		}
	})
}
