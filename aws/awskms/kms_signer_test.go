// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awskms_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"testing"

	"cloudeng.io/aws/awskms"
	"cloudeng.io/aws/awstestutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
)

var awsService *awstestutil.AWS

func TestMain(m *testing.M) {
	awstestutil.AWSTestMain(m, &awsService,
		awstestutil.WithKMS(),
	)
}

func TestSigner(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()

	cfg := awstestutil.DefaultAWSConfig()
	client := awsService.KMS(cfg)

	keyOutput, err := client.CreateKey(ctx, &kms.CreateKeyInput{
		KeyUsage:              types.KeyUsageTypeSignVerify,
		CustomerMasterKeySpec: types.CustomerMasterKeySpecEccNistP256,
		Description:           aws.String("TestSignerKey"),
	})
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	t.Cleanup(func() {
		_, err := client.ScheduleKeyDeletion(context.Background(), &kms.ScheduleKeyDeletionInput{
			KeyId:               keyOutput.KeyMetadata.KeyId,
			PendingWindowInDays: aws.Int32(7),
		})
		if err != nil {
			t.Logf("failed to schedule key deletion: %v", err)
		}
	})

	keyID := aws.ToString(keyOutput.KeyMetadata.KeyId)
	if keyID == "" {
		t.Fatal("keyID is empty")
	}
	t.Logf("Created Key: %s", keyID)

	signer, err := awskms.NewSigner(ctx, client, keyID, string(types.SigningAlgorithmSpecEcdsaSha256))
	if err != nil {
		t.Fatalf("NewSigner failed: %v", err)
	}

	digest := sha256.Sum256([]byte("hello world"))
	signature, err := signer.Sign(rand.Reader, digest[:], nil)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	pubKey := signer.Public()
	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("expected *ecdsa.PublicKey, got %T", pubKey)
	}

	if !ecdsa.VerifyASN1(ecdsaPubKey, digest[:], signature) {
		t.Errorf("signature verification failed")
	}

	digest[3]++
	if ecdsa.VerifyASN1(ecdsaPubKey, digest[:], signature) {
		t.Errorf("signature verification should have failed")
	}
}

type mockKMSClient struct {
	signErr error
}

func (m *mockKMSClient) Sign(ctx context.Context, input *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error) {
	if m.signErr != nil {
		return nil, m.signErr
	}
	return &kms.SignOutput{Signature: []byte("mock-signature")}, nil
}

func (m *mockKMSClient) GetPublicKey(ctx context.Context, input *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)

	return &kms.GetPublicKeyOutput{
		KeyUsage:  types.KeyUsageTypeSignVerify,
		PublicKey: der,
	}, nil
}

func TestSignerErrors(t *testing.T) {
	awstestutil.SkipAWSTests(t)
	ctx := context.Background()
	cfg := awstestutil.DefaultAWSConfig()
	client := awsService.KMS(cfg)

	// 1. Invalid KeyID
	_, err := awskms.NewSigner(ctx, client, "", string(types.SigningAlgorithmSpecEcdsaSha256))
	if err == nil || err.Error() != "awskms.NewSigner: keyID is empty" {
		t.Errorf("expected empty keyID error, got %v", err)
	}

	// 2. Unsupported Algo
	_, err = awskms.NewSigner(ctx, client, "some-key-id", "unsupported-algo")
	if err == nil || err.Error() != "awskms.NewSigner: signingAlgo unsupported-algo is not supported" {
		t.Errorf("expected unsupported algo error, got %v", err)
	}

	// 3. Failing Sign operation
	mock := &mockKMSClient{signErr: fmt.Errorf("mock error")}

	signer, err := awskms.NewSigner(ctx, mock, "mock-key", string(types.SigningAlgorithmSpecEcdsaSha256))
	if err != nil {
		t.Fatalf("setup: failed to create signer with mock client: %v", err)
	}

	digest := sha256.Sum256([]byte("hello world"))
	_, err = signer.Sign(rand.Reader, digest[:], nil)
	if err == nil {
		t.Fatal("expected error from Sign, got nil")
	}
	if want, got := "awskms Sign failed: mock error", err.Error(); want != got {
		t.Errorf("got %v, want %v", got, want)
	}
}
