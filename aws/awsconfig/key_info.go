// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsconfig

import (
	"fmt"

	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/text/textutil"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// KeyInfoExtra is the extra information stored in a key info for AWS.
// It is used to populate the AWS config with the access key ID and region.
// The SecretAccessKey is stored in the token field of the key info.
type KeyInfoExtra struct {
	AccessKeyID string `yaml:"access_key_id"`
	Region      string `yaml:"region"`
}

// ConfigOptionsFromKeyInfo returns the ConfigOptions implied by the key info.
func ConfigOptionsFromKeyInfo(keyInfo keys.Info) ([]ConfigOption, error) {
	var extra KeyInfoExtra
	if err := keyInfo.UnmarshalExtra(&extra); err != nil {
		return []ConfigOption{}, fmt.Errorf("failed to extract %T from key info: %w", extra, err)
	}
	token := keyInfo.Token()
	defer token.Clear()
	provider := credentials.NewStaticCredentialsProvider(
		textutil.TrimUnicodeQuotes(extra.AccessKeyID),
		textutil.TrimUnicodeQuotes(string(token.Value())),
		"")
	return []ConfigOption{
		WithConfigOptions(config.WithRegion(extra.Region)),
		WithConfigOptions(config.WithCredentialsProvider(provider)),
	}, nil
}

// NewKeyInfo creates a new keys.Info appropriate for use with
// static credentials for AWS.
func NewKeyInfo(id, user string, token []byte, extra KeyInfoExtra) keys.Info {
	ki := keys.NewInfo(id, user, token)
	ki.WithExtra(extra)
	return ki
}
