// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsconfig

import (
	"cloudeng.io/cmdutil/keys"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type KeyInfoExtra struct {
	AccessKeyID string `yaml:"access_key_id"`
	Region      string `yaml:"region"`
}

func ConfigOptionsFromKeyInfo(keyInfo keys.Info) []ConfigOption {
	var extra KeyInfoExtra
	if err := keyInfo.ExtraAs(&extra); err != nil {
		return []ConfigOption{}
	}
	token := keyInfo.Token()
	defer token.Clear()
	provider := credentials.NewStaticCredentialsProvider(
		extra.AccessKeyID, string(token.Value()), "")
	return []ConfigOption{
		WithConfigOptions(config.WithRegion(extra.Region)),
		WithConfigOptions(config.WithCredentialsProvider(provider)),
	}
}
