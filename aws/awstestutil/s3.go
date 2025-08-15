// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awstestutil

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyauth "github.com/aws/smithy-go/auth"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type s3AuthSchemeResolver struct {
}

func (s3AuthSchemeResolver) ResolveAuthSchemes(_ context.Context, params *s3.AuthResolverParameters) ([]*smithyauth.Option, error) {
	return []*smithyauth.Option{
		{
			SchemeID: smithyauth.SchemeIDSigV4,
			SignerProperties: func() smithy.Properties {
				var props smithy.Properties
				smithyhttp.SetSigV4SigningName(&props, "s3")
				smithyhttp.SetSigV4SigningRegion(&props, params.Region)
				smithyhttp.SetIsUnsignedPayload(&props, true)
				return props
			}(),
		},
		{
			SchemeID: smithyauth.SchemeIDSigV4A,
			SignerProperties: func() smithy.Properties {
				var props smithy.Properties
				smithyhttp.SetSigV4ASigningName(&props, "s3")
				smithyhttp.SetSigV4ASigningRegions(&props, []string{params.Region})
				return props
			}(),
		},
	}, nil
}

type s3Resolver struct {
	scheme string
	port   string
}

func (r *s3Resolver) ResolveEndpoint(_ context.Context, params s3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	// Use virtual-hosted style endpoints.
	var ep smithyendpoints.Endpoint
	bucket := aws.ToString(params.Bucket)
	if len(bucket) == 0 {
		ep.URI.Host = "s3.localhost.localstack.cloud:" + r.port
	} else {
		ep.URI.Host = aws.ToString(params.Bucket) + ".s3.localhost.localstack.cloud:" + r.port
	}
	ep.URI.Scheme = r.scheme
	return ep, nil
}

func (a *AWS) S3(cfg aws.Config) *s3.Client {
	u := a.uri()
	res := &s3Resolver{scheme: u.Scheme, port: u.Port()}
	opt := s3.WithEndpointResolverV2(res)
	opts := s3.Options{
		AuthSchemeResolver: s3AuthSchemeResolver{},
		Region:             cfg.Region,
		Credentials:        cfg.Credentials,
	}
	return s3.New(opts, opt)
}
