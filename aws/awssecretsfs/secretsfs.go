// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package awssecrets provides an implementation of fs.ReadFileFS that reads
// secrets from the AWS secretsmanager.
package awssecretsfs

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"time"

	"cloudeng.io/aws/awsutil"
	"cloudeng.io/file"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// Option represents an option to New.
type Option func(o *options)

type options struct {
	smOptions           secretsmanager.Options
	recoveryDelayInDays int64
	client              Client
}

// WithSecretsOptions wraps secretsmanager.Options for use when creating an s3.Client.
func WithSecretsOptions(opts ...func(*secretsmanager.Options)) Option {
	return func(o *options) {
		for _, fn := range opts {
			fn(&o.smOptions)
		}
	}
}

// WithRecoveryDelay specifies the number of days to retain a secret after deletion.
// Set to 0 for immediate deletion without recovery, the default is 7 days.
func WithRecoveryDelay(days int64) Option {
	return func(o *options) {
		o.recoveryDelayInDays = days
	}
}

// WithSecretsClient specifies the secretsmanager.Client to use. If not specified, a new is created.
func WithSecretsClient(client Client) Option {
	return func(o *options) {
		o.client = client
	}
}

// New creates a new instance of fs.ReadFile backed by the secretsmanager.
func New(cfg aws.Config, options ...Option) fs.ReadFileFS {
	return NewSecretsFS(cfg, options...)
}

// T implements fs.ReadFileFS for secretsmanager.
type T struct {
	client  Client
	options options
}

// NewSecretsFS creates a new instance of T.
func NewSecretsFS(cfg aws.Config, options ...Option) *T {
	smfs := &T{}
	smfs.options.recoveryDelayInDays = 7
	for _, fn := range options {
		fn(&smfs.options)
	}
	smfs.client = smfs.options.client
	if smfs.client == nil {
		smfs.client = secretsmanager.NewFromConfig(cfg)
	}
	return smfs
}

// Open implements fs.FS. Name can be the short name of the secret or the ARN.
func (smfs *T) Open(name string) (fs.File, error) {
	out, err := readSecret(context.Background(), smfs.client, name)
	if err != nil {
		return nil, translateError(err)
	}
	data := getData(out)
	return &secret{name: aws.ToString(out.Name), size: len(data), buf: bytes.NewBuffer(data)}, nil
}

// ReadFile implements fs.ReadFileFS. Name can be the short name of the secret or the ARN.
func (smfs *T) ReadFile(name string) ([]byte, error) {
	return smfs.readFileCtx(context.Background(), name)
}

// ReadFileCtx is like ReadFile but with a context.
func (smfs *T) ReadFileCtx(ctx context.Context, name string) ([]byte, error) {
	return smfs.readFileCtx(ctx, name)
}

// Delete deletes the secret with the given name. Name can be the short name of the secret or the ARN.
func (smfs *T) Delete(ctx context.Context, nameOrArn string) error {
	arn := nameOrArn
	if !awsutil.IsARN(nameOrArn) {
		arn = getARN(ctx, smfs.client, nameOrArn)
	}
	sin := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(arn),
	}
	if smfs.options.recoveryDelayInDays == 0 {
		sin.ForceDeleteWithoutRecovery = aws.Bool(true)
	} else {
		sin.RecoveryWindowInDays = aws.Int64(smfs.options.recoveryDelayInDays)
	}
	_, err := smfs.client.DeleteSecret(ctx, sin)
	return translateError(err)
}

func (smfs *T) readFileCtx(ctx context.Context, name string) ([]byte, error) {
	out, err := readSecret(ctx, smfs.client, name)
	if err != nil {
		return nil, translateError(err)
	}
	return getData(out), nil
}

type secret struct {
	name string
	size int
	buf  *bytes.Buffer
}

// Stat implements fs.File. The Name and Size fields are populated, the Name
// is the short name of the secret rather than the ARN.
func (s *secret) Stat() (fs.FileInfo, error) {
	return file.NewInfo(s.name, int64(s.size), 0, time.Time{}, nil), nil
}

func (s *secret) Read(buf []byte) (int, error) {
	return s.buf.Read(buf)
}

func (s *secret) Close() error {
	return nil
}

// Client represents the set of AWS Secrets service methods used by awssecretsfs.
type Client interface {
	ListSecretVersionIds(ctx context.Context, params *secretsmanager.ListSecretVersionIdsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretVersionIdsOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}

// getARN returns the ARN for the secret which includes the random string
// created by the secretsmanager rather than the 'short' name of the secret
// so that subsequent operations return ResourceNotFoundException or
// AccessDeniedException cleanly.
func getARN(ctx context.Context, client Client, name string) string {
	out, err := client.ListSecretVersionIds(ctx, &secretsmanager.ListSecretVersionIdsInput{SecretId: aws.String(name)})
	if err != nil {
		return name
	}
	return *out.ARN
}

func getData(out *secretsmanager.GetSecretValueOutput) []byte {
	if out.SecretBinary != nil {
		return out.SecretBinary
	}
	return []byte(aws.ToString(out.SecretString))
}

func readSecret(ctx context.Context, client Client, nameOrArn string) (*secretsmanager.GetSecretValueOutput, error) {
	arn := nameOrArn
	if !awsutil.IsARN(nameOrArn) {
		arn = getARN(ctx, client, nameOrArn)
	}
	return client.GetSecretValue(ctx,
		&secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)})
}

func translateError(err error) error {
	var rnfe *types.ResourceNotFoundException
	if errors.As(err, &rnfe) {
		return fs.ErrNotExist
	}
	return err
}
