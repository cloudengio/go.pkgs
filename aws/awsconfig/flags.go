// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsconfig

import (
	"context"
	"fmt"
	"strings"

	"cloudeng.io/cmdutil/keys"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// AWSFlags defines commonly used flags that control AWS behaviour.
type AWSFlags struct {
	AWS            bool   `subcmd:"aws,false,set to enable AWS functionality" yaml:"aws" doc:"set to true enable AWS functionality"`
	AWSProfile     string `subcmd:"aws-profile,,aws profile to use for config/authentication" yaml:"aws_profile" doc:"aws profile to use for config/authentication"`
	AWSRegion      string `subcmd:"aws-region,,'aws region to use for API calls, overrides the region set in the profile'" yaml:"aws_region" doc:"aws region to use, overrides the region set in the profile"`
	AWSConfigFiles string `subcmd:"aws-config-files,,comma separated list of config files to use in place of those commonly found in $HOME/.aws" yaml:"aws_config_files,flow" doc:"comma separated list of config files to use in place of those commonly found in $HOME/.aws"`
	AWSKeyInfoID   string `subcmd:"aws-key-info-id,,key info ID to use for authentication" yaml:"aws_key_info_id" doc:"key info ID to use for authentication"`
}

// LoadUsingFlags calls awsconfig.Load with options controlled by the
// the specified flags.
func LoadUsingFlags(ctx context.Context, cl AWSFlags) (aws.Config, error) {
	if !cl.AWS {
		return aws.Config{}, fmt.Errorf("aws not enabled")
	}
	opts, err := ConfigOptionsFromFlags(ctx, cl)
	if err != nil {
		return aws.Config{}, err
	}
	return Load(ctx, opts...)
}

// ConfigOptionsFromFlags returns the ConfigOptions implied by the flags.
// NOTE: it always includes config.WithEC2IMDSRegion so that the region
// information is retrieved from EC2 IMDS when it's not found by other
// means.
func ConfigOptionsFromFlags(ctx context.Context, cl AWSFlags) ([]ConfigOption, error) {
	cfg := cl.Config()
	return cfg.Options(ctx)
}

// AWSConfig represents a minimal AWS configuration required to authenticate
// and interact with AWS services.
type AWSConfig struct {
	AWS            bool     `yaml:"aws"`
	AWSProfile     string   `yaml:"aws_profile"`
	AWSRegion      string   `yaml:"aws_region"`
	AWSConfigFiles []string `yaml:"aws_config_files"`
	AWSKeyInfoID   string   `yaml:"aws_key_info_id"`
}

// Config converts the flags to a AWSConfig instance.
func (c AWSFlags) Config() AWSConfig {
	var files []string
	if c.AWSConfigFiles != "" {
		files = strings.Split(c.AWSConfigFiles, ",")
	}
	return AWSConfig{
		AWS:            c.AWS,
		AWSProfile:     c.AWSProfile,
		AWSRegion:      c.AWSRegion,
		AWSConfigFiles: files,
		AWSKeyInfoID:   c.AWSKeyInfoID,
	}
}

// Load calls awsconfig.Load with options controlled by the config.
func (c AWSConfig) Load(ctx context.Context) (aws.Config, error) {
	if !c.AWS {
		return aws.Config{}, fmt.Errorf("aws not enabled")
	}
	opts, err := c.Options(ctx)
	if err != nil {
		return aws.Config{}, err
	}
	return Load(ctx, opts...)
}

// Options returns the ConfigOptions implied by the config.
// NOTE: it always includes config.WithEC2IMDSRegion so that the region
// information is retrieved from EC2 IMDS when it's not found by other
// means.
func (c AWSConfig) Options(ctx context.Context) ([]ConfigOption, error) {
	if !c.AWS {
		return nil, nil
	}
	opts := []ConfigOption{}
	if len(c.AWSConfigFiles) > 0 {
		opts = append(opts, WithConfigOptions(config.WithSharedConfigFiles(c.AWSConfigFiles)))
	}
	if len(c.AWSProfile) > 0 {
		opts = append(opts,
			WithConfigOptions(config.WithSharedConfigProfile(c.AWSProfile)))
	}
	if len(c.AWSRegion) > 0 {
		opts = append(opts,
			WithConfigOptions(config.WithRegion(c.AWSRegion)))
	}
	if len(c.AWSKeyInfoID) > 0 {
		if ki, ok := keys.KeyInfoFromContextForID(ctx, c.AWSKeyInfoID); ok {
			co, err := ConfigOptionsFromKeyInfo(ki)
			if err != nil {
				return nil, err
			}
			opts = append(opts, co...)
		} else {
			return nil, fmt.Errorf("key info ID %q not found", c.AWSKeyInfoID)
		}
	}
	opts = append(opts, WithConfigOptions(
		config.WithEC2IMDSRegion(),
	))
	return opts, nil
}
