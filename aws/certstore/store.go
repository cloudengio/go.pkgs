// Package certstore provides an implementation of a autocert.DirCache
// and cloudeng.io/webapp.CertStore for use when managing TLS certificates
// on AWS. In particular, it uses the AWS secrets manager to store TLS
// certificates.
package certstore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webauth/acme"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/crypto/acme/autocert"
)

var (
	accountIDOnce sync.Once
	accountID     string
	accountIDErr  error
)

func getAccountIDOnce(ctx context.Context, cfg aws.Config) (string, error) {
	accountIDOnce.Do(func() {
		accountID, accountIDErr = awsconfig.AccountID(ctx, cfg)
	})
	return accountID, accountIDErr
}

// ErrUnsupportedOperation is returned for any unsupported operations.
var ErrUnsupportedOperation = errors.New("unsupported operation")

type dircache struct {
	localCache autocert.Cache
	awsCache   autocert.Cache
}

func isLocal(name string) bool {
	return strings.HasSuffix(name, "+token") ||
		strings.HasSuffix(name, "+rsa") ||
		name == "acme_account+key"
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
	//	region    string
	config    aws.Config
	hasConfig bool
	//keys      map[string]string
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
	accountID, err := awsconfig.AccountID(ctx, ac.config)
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

// Delete implements autocert.Cache.
// Note that currently deletions of aws stored keys are not allowed.
func (ac *awscache) Delete(ctx context.Context, name string) error {
	return ErrUnsupportedOperation
}

// Get implements autocert.Cache.
func (ac *awscache) Get(ctx context.Context, name string) ([]byte, error) {
	client, err := ac.newClient(ctx, name, true)
	if err != nil {
		return nil, err
	}
	args := secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	}
	// NOTE must have kms:Decrypt permission for this key.
	results, err := client.GetSecretValue(ctx, &args)
	if err != nil {
		return nil, err
	}
	data := make([]byte, len(results.SecretBinary))
	copy(data, results.SecretBinary)
	log.Printf("results: %v", results)
	return data, nil
}

// Put implements autocert.Cache.
func (ac *awscache) Put(ctx context.Context, name string, data []byte) error {
	client, err := ac.newClient(ctx, name, false)
	if err != nil {
		return err
	}
	args := secretsmanager.UpdateSecretInput{
		SecretId: aws.String(name),
		//KmsKeyId:     ac.kmsKeyFor(name),
		SecretBinary: data,
	}
	results, err := client.UpdateSecret(ctx, &args)
	log.Printf("results: %v", results)
	return err
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
func (f CertStoreFactory) New(ctx context.Context, name string, opts ...interface{}) (webapp.CertStore, error) {
	switch f.typ {
	case awsCacheName:
		return NewAWSCache(captureOpts(opts)...), nil
	}
	return nil, errors.New(unsupported(f.typ))
}

// Describe implements webapp.CertStoreFactory.
func (f CertStoreFactory) Describe() string {
	switch f.typ {
	case awsCacheName:
		return awsCacheName + " retrieves certificates from AWS secretsmanager"
	}
	panic(unsupported(f.typ))
}
