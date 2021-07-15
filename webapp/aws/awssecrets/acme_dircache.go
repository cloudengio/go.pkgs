package awssecrets

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webauth/acme"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"golang.org/x/crypto/acme/autocert"
)

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

func NewHybridCache(dir, region string, opts ...AWSCacheOption) autocert.Cache {
	localCache := acme.NewDirCache(dir, false)
	awsCache := NewAWSCache(region)
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
	region    string
	config    aws.Config
	useConfig bool
	keys      map[string]string
}

type AWSCacheOption func(a *awscache)

func AddKMSKey(name, key string) AWSCacheOption {
	return func(a *awscache) {
		a.keys[name] = key
	}
}

func SetAWSConfig(cfg *aws.Config) AWSCacheOption {
	return func(a *awscache) {
		a.config = cfg.Copy()
		a.useConfig = true
	}
}

func NewAWSCache(region string, opts ...AWSCacheOption) autocert.Cache {
	ac := &awscache{region: region}
	for _, fn := range opts {
		fn(ac)
	}
	return ac
}

func (ac *awscache) newClient() *secretsmanager.Client {
	if ac.useConfig {
		return secretsmanager.NewFromConfig(ac.config)
	}
	opts := secretsmanager.Options{Region: ac.region}
	return secretsmanager.New(opts)
}

func (ac *awscache) kmsKeyFor(name string) *string {
	if id, ok := ac.keys[name]; ok {
		return aws.String(id)
	}
	return nil
}

// Delete implements autocert.Cache.
// Note that currently deletions of aws stored keys are not allowed.
func (ac *awscache) Delete(ctx context.Context, name string) error {
	return ErrUnsupportedOperation
}

// Get implements autocert.Cache.
func (ac *awscache) Get(ctx context.Context, name string) ([]byte, error) {
	client := ac.newClient()
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
	client := ac.newClient()
	args := secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(name),
		KmsKeyId:     ac.kmsKeyFor(name),
		SecretBinary: data,
	}
	results, err := client.UpdateSecret(ctx, &args)
	log.Printf("results: %v", results)
	return err
}

const (
	awsCacheName = "autocert-aws-secrets-cache"
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
		return NewAWSCache(name, captureOpts(opts)...), nil
	}
	return nil, errors.New(unsupported(f.typ))
}
