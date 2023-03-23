// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package awscertstore provides an implementation of a autocert.DirCache
// and cloudeng.io/webapp.CertStore for use when managing TLS certificates
// on AWS. In particular, it uses the AWS secrets manager to store TLS
// certificates.
package awscertstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloudeng.io/aws/awsutil"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webauth/acme"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/crypto/acme/autocert"
)

var (
	// ErrUnsupportedOperation is returned for any unsupported operations.
	ErrUnsupportedOperation = errors.New("unsupported operation")
	// ErrCacheMiss is the same as autocert.ErrCacheMiss
	ErrCacheMiss = autocert.ErrCacheMiss
)

type dircache struct {
	localCache autocert.Cache
	awsCache   autocert.Cache
}

func isLocal(name string) bool {
	return strings.HasSuffix(name, "+token") ||
		strings.HasSuffix(name, "+rsa") ||
		strings.Contains(name, "http-01") ||
		(strings.HasPrefix(name, "acme_account") &&
			strings.HasSuffix(name, "key"))
}

// NewHybridCache returns an instance of autocert.Cache that will store
// certificates in 'backing' store, but use the local file system for
// temporary/private data such as the ACME client's private key. This
// allows for certificates to be shared across multiple hosts by using
// a distributed 'backing' store such as AWS' secretsmanager.
func NewHybridCache(dir string, opts ...AWSCacheOption) autocert.Cache {
	localCache := acme.NewDirCache(dir, false)
	awsCache := NewAWSCache(opts...)
	return &dircache{
		localCache: localCache,
		awsCache:   awsCache,
	}
}

// Delete implements autocert.Cache.
func (dc *dircache) Delete(ctx context.Context, name string) error {
	if isLocal(name) {
		return dc.localCache.Delete(ctx, name)
	}
	return dc.awsCache.Delete(ctx, name)
}

// Get implements autocert.Cache.
func (dc *dircache) Get(ctx context.Context, name string) ([]byte, error) {
	if isLocal(name) {
		return dc.localCache.Get(ctx, name)
	}
	return dc.awsCache.Get(ctx, name)
}

// Put implements autocert.Cache.
func (dc *dircache) Put(ctx context.Context, name string, data []byte) error {
	if isLocal(name) {
		return dc.localCache.Put(ctx, name, data)
	}
	return dc.awsCache.Put(ctx, name, data)
}

type awscache struct {
	config    aws.Config
	hasConfig bool
}

// AWSCacheOption represents an option to NewAWSCache.
type AWSCacheOption func(a *awscache)

// WithAWSConfig specifies the aws.Config to use, it must be used
// to specify the aws.Config to use for operations on the underlying
// secrets manager.
func WithAWSConfig(cfg aws.Config) AWSCacheOption {
	return func(a *awscache) {
		a.config = cfg.Copy()
		a.hasConfig = true
	}
}

// NewAWSCache returns an instance of autocert.Cache that uses the
// AWS secretsmanager. It assumes that a secret has already been
// created for storing a given certificate and that the name of
// the certificate is the same as the name of the secret.
func NewAWSCache(opts ...AWSCacheOption) autocert.Cache {
	ac := &awscache{}
	for _, fn := range opts {
		fn(ac)
	}
	return ac
}

func (ac *awscache) newClient(ctx context.Context, name string, readonly bool) (*secretsmanager.Client, error) {
	if !ac.hasConfig {
		return nil, fmt.Errorf("no aws.Config was specified, use WithAWSConfig when creating the store")
	}
	accountID, err := awsutil.AccountID(ctx, ac.config)
	if err != nil {
		return nil, err
	}
	stsSvc := sts.NewFromConfig(ac.config)
	suffix := "readWrite"
	if readonly {
		suffix = "readOnly"
	}
	arn := fmt.Sprintf("arn:aws:iam::%v:role/secret_%v_%v", accountID, name, suffix)
	creds := stscreds.NewAssumeRoleProvider(stsSvc, arn)
	cfg := ac.config.Copy()
	cfg.Credentials = aws.NewCredentialsCache(creds)
	return secretsmanager.NewFromConfig(cfg), nil
}

// make sure to return an ErrCacheMiss error when a certificate is
// not currently in the store since the calling autocert code tests
// for this specific error.
func translateError(err error) error {
	var notFound *types.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return ErrCacheMiss
	}
	return err
}

// Delete implements autocert.Cache.
// Note that currently deletions of aws stored keys are not allowed.
func (ac *awscache) Delete(_ context.Context, name string) error {
	name = strings.ReplaceAll(name, ".", "_")
	_ = name
	return ErrUnsupportedOperation
}

// getARN returns the ARN for the secret which includes the random string
// created by the secretsmanager rather than the 'short' name of the secret
// so that subsequent operations return ResourceNotFoundException or
// AccessDeniedException cleanly.
func getARN(ctx context.Context, client *secretsmanager.Client, name string) string {
	out, err := client.ListSecretVersionIds(ctx, &secretsmanager.ListSecretVersionIdsInput{SecretId: aws.String(name)})
	if err != nil {
		return name
	}
	return *out.ARN
}

// Get implements autocert.Cache.
func (ac *awscache) Get(ctx context.Context, name string) ([]byte, error) {
	name = strings.ReplaceAll(name, ".", "_")
	client, err := ac.newClient(ctx, name, true)
	if err != nil {
		return nil, err
	}
	arn := getARN(ctx, client, name)
	args := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	}
	results, err := client.GetSecretValue(ctx, &args)
	if err != nil {
		return nil, translateError(err)
	}
	data := make([]byte, len(results.SecretBinary))
	copy(data, results.SecretBinary)
	return data, nil
}

// Put implements autocert.Cache.
func (ac *awscache) Put(ctx context.Context, name string, data []byte) error {
	name = strings.ReplaceAll(name, ".", "_")
	client, err := ac.newClient(ctx, name, false)
	if err != nil {
		return err
	}
	arn := getARN(ctx, client, name)
	args := secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(arn),
		SecretBinary: data,
	}
	_, err = client.UpdateSecret(ctx, &args)
	return translateError(err)
}

const (
	awsCacheName = "autocert-aws-secrets-cache"
)

var (
	// AutoCertStore creates instances of webapp.CertStore using
	// NewHybridCache.
	AutoCertStore = CertStoreFactory{awsCacheName}
)

// CertStoreFactory represents the webapp.CertStore's that can be
// created by this package.
type CertStoreFactory struct {
	typ string
}

// Type implements webapp.CertStoreFactory.
func (f CertStoreFactory) Type() string {
	return f.typ
}

func unsupported(typ string) string {
	return fmt.Sprintf(
		"unsupported factory type: %s: use one of %s", typ, strings.Join([]string{awsCacheName}, ","))
}

func captureOpts(opts []interface{}) []AWSCacheOption {
	var awsOpts []AWSCacheOption
	for _, o := range opts {
		if awsOpt, ok := o.(AWSCacheOption); ok {
			awsOpts = append(awsOpts, awsOpt)
		}
	}
	return awsOpts
}

// New implements webapp.CertStoreFactory.
func (f CertStoreFactory) New(_ context.Context, _ string, opts ...interface{}) (webapp.CertStore, error) {
	if f.typ == awsCacheName {
		return NewAWSCache(captureOpts(opts)...), nil
	}
	return nil, errors.New(unsupported(f.typ))
}

// Describe implements webapp.CertStoreFactory.
func (f CertStoreFactory) Describe() string {
	if f.typ == awsCacheName {
		return awsCacheName + " retrieves certificates from AWS secretsmanager"
	}
	panic(unsupported(f.typ))
}
