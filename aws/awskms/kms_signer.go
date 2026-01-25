// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awskms

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

const defaultTimeout = time.Second * 5

type Client interface {
	Sign(ctx context.Context, input *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, input *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
}

type Signer struct {
	client Client
	keyID  string
	algo   kmstypes.SigningAlgorithmSpec
	pubKey crypto.PublicKey
}

func (s *Signer) Sign(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
	req := &kms.SignInput{
		KeyId:            aws.String(s.keyID),
		SigningAlgorithm: s.algo,
		Message:          digest,
		MessageType:      kmstypes.MessageTypeDigest,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	resp, err := s.client.Sign(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("awskms Sign failed: %w", err)
	}

	return resp.Signature, nil
}

func getPublicKey(ctx context.Context, client Client, keyID string) (crypto.PublicKey, error) {

	input := kms.GetPublicKeyInput{
		KeyId: aws.String(keyID),
	}
	output, err := client.GetPublicKey(ctx, &input)
	if err != nil {
		return nil, fmt.Errorf(`failed to get public key from KMS: %w`, err)
	}

	if output.KeyUsage != kmstypes.KeyUsageTypeSignVerify {
		return nil, fmt.Errorf(`invalid key usage. expected SIGN_VERIFY, got %q`, output.KeyUsage)
	}

	key, err := x509.ParsePKIXPublicKey(output.PublicKey)
	if err != nil {
		return nil, fmt.Errorf(`failed to parse key: %w`, err)
	}

	return key, nil
}

func (s *Signer) Public() crypto.PublicKey {
	return s.pubKey
}

var supportedAlgoSpecs []kmstypes.SigningAlgorithmSpec

func init() {
	var spec kmstypes.SigningAlgorithmSpec
	supportedAlgoSpecs = spec.Values()
}

func NewSigner(ctx context.Context, client Client, keyID, signingAlgo string) (crypto.Signer, error) {
	if len(keyID) == 0 {
		return nil, fmt.Errorf("awskms.NewSigner: keyID is empty")
	}
	algo := kmstypes.SigningAlgorithmSpec(signingAlgo)
	if !slices.Contains(supportedAlgoSpecs, algo) {
		return nil, fmt.Errorf("awskms.NewSigner: signingAlgo %v is not supported", signingAlgo)
	}
	s := &Signer{
		client: client,
		keyID:  keyID,
		algo:   algo,
	}
	pk, err := getPublicKey(ctx, s.client, keyID)
	if err != nil {
		return nil, err
	}
	s.pubKey = pk
	return s, nil
}

func PublicKey(ctx context.Context, client Client, keyID string) (crypto.PublicKey, error) {
	return getPublicKey(ctx, client, keyID)
}
