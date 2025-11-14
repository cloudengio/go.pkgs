// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// AWSFlags defines commonly used flags that control AWS behaviour.
type AWSFlags struct {
	AWS            bool   `subcmd:"aws,false,set to enable AWS functionality" yaml:"aws" cmd:"set to true enable AWS functionality"`
	AWSProfile     string `subcmd:"aws-profile,,aws profile to use for config/authentication" yaml:"aws_profile" cmd:"aws profile to use for config/authentication"`
	AWSRegion      string `subcmd:"aws-region,,'aws region to use for API calls, overrides the region set in the profile'" yaml:"aws_region" cmd:"aws region to use, overrides the region set in the profile"`
	AWSConfigFiles string `subcmd:"aws-config-files,,comma separated list of config files to use in place of those commonly found in $HOME/.aws" yaml:"aws_config_files,flow" cmd:"comma separated list of config files to use in place of those commonly found in $HOME/.aws"`
}

// LoadUsingFlags calls awsconfig.Load with options controlled by the
// the specified flags.
func LoadUsingFlags(ctx context.Context, cl AWSFlags) (aws.Config, error) {
	if !cl.AWS {
		return aws.Config{}, fmt.Errorf("aws not enabled")
	}
	return Load(ctx, ConfigOptionsFromFlags(cl)...)
}

// ConfigOptionsFromFlags returns the ConfigOptions implied by the flags.
// NOTE: it always includes config.WithEC2IMDSRegion so that the region
// information is retrieved from EC2 IMDS when it's not found by other
// means.
func ConfigOptionsFromFlags(cl AWSFlags) []ConfigOption {
	cfg := cl.Config()
	return cfg.Options()
}

// AWSConfig represents a minimal AWS configuration required to authenticate
// and interact with AWS services.
type AWSConfig struct {
	AWS            bool     `yaml:"aws"`
	AWSProfile     string   `yaml:"aws_profile"`
	AWSRegion      string   `yaml:"aws_region"`
	AWSConfigFiles []string `yaml:"aws_config_files"`
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
	}
}

// Load calls awsconfig.Load with options controlled by the config.
func (c AWSConfig) Load(ctx context.Context) (aws.Config, error) {
	if !c.AWS {
		return aws.Config{}, fmt.Errorf("aws not enabled")
	}
	return Load(ctx, c.Options()...)
}

// Options returns the ConfigOptions implied by the config.
// NOTE: it always includes config.WithEC2IMDSRegion so that the region
// information is retrieved from EC2 IMDS when it's not found by other
// means.
func (c AWSConfig) Options() []ConfigOption {
	if !c.AWS {
		return nil
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
	opts = append(opts, WithConfigOptions(
		config.WithEC2IMDSRegion(),
	))
	return opts
}
