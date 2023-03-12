// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package awstokens provides a very simple mechanism for retrieving
// secrets stored with the AWS secretsmanager service.
package awstokens

import (
	"context"

	"cloudeng.io/aws/awsutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

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

// GetSecret returns the value of the secret with the given name or arn.
func GetSecret(ctx context.Context, config aws.Config, nameOrArn string) (string, error) {
	client := secretsmanager.NewFromConfig(config)
	arn := nameOrArn
	if !awsutil.IsARN(nameOrArn) {
		arn = getARN(ctx, client, nameOrArn)
	}
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)})
	if err != nil {
		return "", err
	}
	return *out.SecretString, nil
}
