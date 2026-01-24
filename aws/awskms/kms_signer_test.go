// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awskms_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
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

	digest[3] = digest[3] + 1
	if ecdsa.VerifyASN1(ecdsaPubKey, digest[:], signature) {
		t.Errorf("signature verification should have failed")
	}
}
