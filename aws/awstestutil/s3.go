// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awstestutil

import (
	"context"
	"net/url"

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
	uri url.URL
}

func (r *s3Resolver) ResolveEndpoint(_ context.Context, params s3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	var ep smithyendpoints.Endpoint
	ep.URI = r.uri
	ep.URI.Path = aws.ToString(params.Bucket)
	return ep, nil
}

func (a *AWS) S3(cfg aws.Config) *s3.Client {
	res := &s3Resolver{a.uri()}
	opt := s3.WithEndpointResolverV2(res)
	opts := s3.Options{
		AuthSchemeResolver: s3AuthSchemeResolver{},
		UsePathStyle:       true,
		Region:             cfg.Region,
		Credentials:        cfg.Credentials,
	}
	return s3.New(opts, opt)
}
