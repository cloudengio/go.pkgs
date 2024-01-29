// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package awsutil provides support for testing AWS packages and applications.
package awstestutil

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"github.com/orlangure/gnomock"
	"github.com/orlangure/gnomock/preset/localstack"
)

type Option func(o *Options)

type Service string

const (
	S3             Service = Service(localstack.S3)
	SecretsManager Service = Service(localstack.SecretsManager)
)

func withoutGnomock(m *testing.M) {
	os.Exit(m.Run())
}

func withGnomock(m *testing.M, service **AWS, opts []Option) {
	svc := NewLocalAWS(opts...)
	if err := svc.Start(); err != nil {
		panic(fmt.Sprintf("failed to start aws test services: %v", err))
	}
	*service = svc
	code := m.Run()
	if code != 0 {
		svc.Stop() //nolint: errcheck
		os.Exit(code)
	}
	svc.Stop() //nolint: errcheck
}

func isOnGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") != ""
}

func WithDebug(log io.Writer) Option {
	return func(o *Options) {
		o.gnomockOptions = append(o.gnomockOptions,
			gnomock.WithDebugMode(),
			gnomock.WithLogWriter(log),
			gnomock.WithEnv("DEBUG=1"),
			gnomock.WithEnv("LS_LOG=trace"))
	}
}

func WithS3() Option {
	return func(o *Options) {
		o.localStackServices = append(o.localStackServices, localstack.S3)
	}
}

func WithSecretsManager() Option {
	return func(o *Options) {
		o.localStackServices = append(o.localStackServices, localstack.SecretsManager)
	}
}

// WithS3Tree configures the local S3 instance with the contents of the
// specified directory. The first level of directories under dir are used as
// bucket names, the second and deeper levels as prefixes and objects within
// those buckets etc.
func WithS3Tree(dir string) Option {
	return func(o *Options) {
		o.localStackServices = append(o.localStackServices, localstack.S3)
		o.localStackoptions = append(o.localStackoptions, localstack.WithS3Files(dir))
	}
}

type Options struct {
	localStackServices []localstack.Service
	localStackoptions  []localstack.Option
	gnomockOptions     []gnomock.Option
}

type AWS struct {
	mu        sync.Mutex
	started   bool
	container *gnomock.Container
	options   Options
}

func NewLocalAWS(opts ...Option) *AWS {
	a := &AWS{}
	for _, fn := range opts {
		fn(&a.options)
	}
	return a
}

func (a *AWS) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.started {
		return nil
	}
	localStackOpts := append(
		[]localstack.Option{
			localstack.WithServices(a.options.localStackServices...)},
		a.options.localStackoptions...)
	preset := localstack.Preset(localStackOpts...)
	c, err := gnomock.Start(preset, a.options.gnomockOptions...)
	if err != nil {
		return err
	}
	a.started = true
	a.container = c
	return nil
}

func (a *AWS) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.started {
		return nil
	}
	err := gnomock.Stop(a.container)
	a.started = false
	a.container = nil
	return err
}

// hostOnlyResolver is used for services that only require a host name in
// the service endpoint/URI.
type hostOnlyResolver[T any] struct {
	ep smithyendpoints.Endpoint
}

func (r hostOnlyResolver[T]) ResolveEndpoint(_ context.Context, _ T) (smithyendpoints.Endpoint, error) {
	return r.ep, nil
}

func newHostOnlyResolver[T any](u url.URL) hostOnlyResolver[T] {
	var r hostOnlyResolver[T]
	r.ep.URI = u
	return r
}

func (a *AWS) uri() url.URL {
	var u url.URL
	u.Scheme = "http"
	u.Host = a.container.Address(localstack.APIPort)
	return u
}

func (a *AWS) SecretsManager(cfg aws.Config) *secretsmanager.Client {
	res := newHostOnlyResolver[secretsmanager.EndpointParameters](a.uri())
	opt := secretsmanager.WithEndpointResolverV2(res)
	return secretsmanager.NewFromConfig(cfg, opt)
}

func DefaultAWSConfig() aws.Config {
	return aws.Config{
		Region:      "us-west-2",
		Credentials: credentials.NewStaticCredentialsProvider("a", "b", "c"),
	}
}
